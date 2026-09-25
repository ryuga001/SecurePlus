package scanner

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"dpdp-backend/internal/delivery/services/inspection"
)

const (
	SpoolPattern = "dpdp-scan-*"

	readBufferSize  = 64 << 10
	spoolBufferSize = 1 << 20
)

var (
	ErrFileTooLarge  = errors.New("file exceeds the scanner size limit")
	ErrLimitExceeded = errors.New("file exceeds the scanner extraction limit")
	ErrParse         = errors.New("file could not be parsed")
)

var (
	utf8BOM    = []byte{0xEF, 0xBB, 0xBF}
	utf16LEBOM = []byte{0xFF, 0xFE}
	utf16BEBOM = []byte{0xFE, 0xFF}
)

var mimeExtensions = map[string]string{
	"text/plain":             "txt",
	"text/csv":               "csv",
	"application/csv":        "csv",
	"application/json":       "json",
	"text/json":              "json",
	"application/xml":        "xml",
	"text/xml":               "xml",
	"application/javascript": "js",
	"text/javascript":        "js",
	"application/pdf":        "pdf",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   "docx",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "xlsx",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",
}

var (
	readBuffers  = sync.Pool{New: func() any { buffer := make([]byte, readBufferSize); return &buffer }}
	spoolBuffers = sync.Pool{New: func() any { buffer := make([]byte, spoolBufferSize); return &buffer }}
)

type Processor interface {
	Extract(ctx context.Context, r io.Reader, w io.Writer) error
}

type Limits struct {
	SpoolDir          string
	MaxSpoolBytes     int64
	MaxExtractedBytes int64
	MaxPDFBytes       int64
	MaxZipDirectory   int64
}

func DefaultLimits(spoolDir string, maxSpoolBytes int64) Limits {
	return Limits{
		SpoolDir:          spoolDir,
		MaxSpoolBytes:     maxSpoolBytes,
		MaxExtractedBytes: 2 << 30,
		MaxPDFBytes:       128 << 20,
		MaxZipDirectory:   1 << 20,
	}
}

type Processors struct {
	byExtension map[string]Processor
}

func NewProcessors(items map[string]Processor) *Processors {
	registry := make(map[string]Processor, len(items))
	for extension, processor := range items {
		registry[strings.ToLower(extension)] = processor
	}

	return &Processors{byExtension: registry}
}

func DefaultProcessors(limits Limits) *Processors {
	text := TextProcessor{}
	office := OfficeProcessor{Limits: limits}
	document := PDFProcessor{Limits: limits}

	return NewProcessors(map[string]Processor{
		"txt":  text,
		"csv":  text,
		"json": text,
		"xml":  text,
		"js":   text,
		"docx": office,
		"xlsx": office,
		"pptx": office,
		"pdf":  document,
	})
}

func (p *Processors) For(extension string) (Processor, bool) {
	processor, ok := p.byExtension[extension]

	return processor, ok
}

func (p *Processors) Extensions() []string {
	extensions := make([]string, 0, len(p.byExtension))
	for extension := range p.byExtension {
		extensions = append(extensions, extension)
	}

	slices.Sort(extensions)

	return extensions
}

func ResolveExtension(name, mimeType string) string {
	if extension := inspection.ExtensionOf(name); extension != "" {
		return extension
	}

	media, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		return ""
	}

	return mimeExtensions[media]
}

type TextProcessor struct{}

func (TextProcessor) Extract(ctx context.Context, r io.Reader, w io.Writer) error {
	buffered := bufio.NewReaderSize(r, readBufferSize)

	head, err := buffered.Peek(3)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return err
	}

	var reader io.Reader = buffered

	switch {
	case bytes.HasPrefix(head, utf8BOM):
		if _, err := buffered.Discard(len(utf8BOM)); err != nil {
			return err
		}
	case bytes.HasPrefix(head, utf16LEBOM), bytes.HasPrefix(head, utf16BEBOM):
		reader = transform.NewReader(buffered, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())
	}

	return copyContext(ctx, w, reader)
}

func copyContext(ctx context.Context, w io.Writer, r io.Reader) error {
	buffer := readBuffers.Get().(*[]byte)
	defer readBuffers.Put(buffer)

	return copyWith(ctx, w, r, *buffer)
}

func copyWith(ctx context.Context, w io.Writer, r io.Reader, buffer []byte) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		n, err := r.Read(buffer)
		if n > 0 {
			if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
		}

		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err
		}
	}
}

type spooled struct {
	*os.File
	size int64
}

func spool(ctx context.Context, r io.Reader, dir string, limit int64) (*spooled, error) {
	file, err := os.CreateTemp(dir, SpoolPattern)
	if err != nil {
		return nil, fmt.Errorf("spool file: %w", err)
	}

	buffer := spoolBuffers.Get().(*[]byte)
	defer spoolBuffers.Put(buffer)

	counter := &countingWriter{w: file}

	if err := copyWith(ctx, counter, io.LimitReader(r, limit+1), *buffer); err != nil {
		discard(&spooled{File: file})

		return nil, err
	}

	if counter.n > limit {
		discard(&spooled{File: file})

		return nil, ErrFileTooLarge
	}

	return &spooled{File: file, size: counter.n}, nil
}

func discard(file *spooled) {
	if file == nil || file.File == nil {
		return
	}

	name := file.Name()
	_ = file.Close()
	_ = os.Remove(name)
}

func RemoveStaleSpools(dir string) (int, error) {
	matches, err := filepath.Glob(filepath.Join(dir, SpoolPattern))
	if err != nil {
		return 0, err
	}

	removed := 0
	for _, match := range matches {
		if os.Remove(match) == nil {
			removed++
		}
	}

	return removed, nil
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)

	return n, err
}
