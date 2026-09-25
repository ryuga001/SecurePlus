package scanner

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	eocdSignature  = 0x06054b50
	eocdLength     = 22
	eocdSearchSpan = 64<<10 + eocdLength

	tagNameLimit    = 32
	entityLimit     = 12
	textFlushBuffer = 32 << 10
)

type OfficeProcessor struct {
	Limits Limits
}

func (p OfficeProcessor) Extract(ctx context.Context, r io.Reader, w io.Writer) error {
	file, err := spool(ctx, r, p.Limits.SpoolDir, p.Limits.MaxSpoolBytes)
	if err != nil {
		return err
	}
	defer discard(file)

	if err := checkZipDirectory(file, file.size, p.Limits.MaxZipDirectory); err != nil {
		return err
	}

	archive, err := zip.NewReader(file, file.size)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrParse, err)
	}

	budget := &budgetReader{remaining: p.Limits.MaxExtractedBytes}
	extractor := &xmlText{w: w, out: make([]byte, 0, textFlushBuffer)}

	for _, part := range archive.File {
		if !officePart(part.Name) {
			continue
		}

		if err := p.extractPart(ctx, part, budget, extractor); err != nil {
			return err
		}
	}

	return nil
}

func (p OfficeProcessor) extractPart(ctx context.Context, part *zip.File, budget *budgetReader, extractor *xmlText) error {
	content, err := part.Open()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrParse, err)
	}
	defer content.Close()

	budget.r = content
	extractor.reset()

	buffer := readBuffers.Get().(*[]byte)
	defer readBuffers.Put(buffer)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		n, err := budget.Read(*buffer)
		if n > 0 {
			if writeErr := extractor.feed((*buffer)[:n]); writeErr != nil {
				return writeErr
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			if errors.Is(err, ErrLimitExceeded) || ctx.Err() != nil {
				return err
			}

			return fmt.Errorf("%w: %v", ErrParse, err)
		}
	}

	return extractor.finish()
}

func officePart(name string) bool {
	if !strings.HasSuffix(name, ".xml") {
		return false
	}

	switch {
	case name == "word/document.xml",
		name == "word/footnotes.xml",
		name == "word/endnotes.xml",
		name == "word/comments.xml",
		name == "xl/sharedStrings.xml":
		return true
	}

	for _, prefix := range []string{
		"word/header",
		"word/footer",
		"xl/worksheets/sheet",
		"xl/comments",
		"ppt/slides/slide",
		"ppt/notesSlides/notesSlide",
	} {
		if strings.HasPrefix(name, prefix) && !strings.Contains(name[len(prefix):], "/") {
			return true
		}
	}

	return false
}

func checkZipDirectory(file io.ReaderAt, size, limit int64) error {
	span := min(size, int64(eocdSearchSpan))
	if span < eocdLength {
		return fmt.Errorf("%w: archive too short", ErrParse)
	}

	tail := make([]byte, span)
	if _, err := file.ReadAt(tail, size-span); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: %v", ErrParse, err)
	}

	for index := len(tail) - eocdLength; index >= 0; index-- {
		if binary.LittleEndian.Uint32(tail[index:]) != eocdSignature {
			continue
		}

		directorySize := binary.LittleEndian.Uint32(tail[index+12:])
		if directorySize == 0xFFFFFFFF || int64(directorySize) > limit {
			return ErrLimitExceeded
		}

		return nil
	}

	return fmt.Errorf("%w: missing end of central directory", ErrParse)
}

type budgetReader struct {
	r         io.Reader
	remaining int64
}

func (b *budgetReader) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		return 0, ErrLimitExceeded
	}

	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}

	n, err := b.r.Read(p)
	b.remaining -= int64(n)

	return n, err
}

type xmlState uint8

const (
	stateText xmlState = iota
	stateTag
	stateEntity
	stateComment
	stateCData
)

type xmlText struct {
	w   io.Writer
	out []byte

	state    xmlState
	name     []byte
	naming   bool
	quote    byte
	entity   []byte
	dashes   int
	brackets int
}

func (x *xmlText) reset() {
	x.state = stateText
	x.name = x.name[:0]
	x.naming = false
	x.quote = 0
	x.entity = x.entity[:0]
	x.dashes = 0
	x.brackets = 0
}

func (x *xmlText) feed(data []byte) error {
	for _, character := range data {
		switch x.state {
		case stateText:
			x.text(character)
		case stateTag:
			x.tag(character)
		case stateEntity:
			x.entityByte(character)
		case stateComment:
			x.comment(character)
		case stateCData:
			x.cdata(character)
		}

		if len(x.out) >= textFlushBuffer-utf8.UTFMax-entityLimit {
			if err := x.flush(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (x *xmlText) finish() error {
	if x.state == stateEntity {
		x.out = append(x.out, '&')
		x.out = append(x.out, x.entity...)
	}

	x.out = append(x.out, '\n')
	x.reset()

	return x.flush()
}

func (x *xmlText) flush() error {
	if len(x.out) == 0 {
		return nil
	}

	_, err := x.w.Write(x.out)
	x.out = x.out[:0]

	return err
}

func (x *xmlText) text(character byte) {
	switch character {
	case '<':
		x.state = stateTag
		x.name = x.name[:0]
		x.naming = true
		x.quote = 0
	case '&':
		x.state = stateEntity
		x.entity = x.entity[:0]
	default:
		x.out = append(x.out, character)
	}
}

func (x *xmlText) tag(character byte) {
	if x.quote != 0 {
		if character == x.quote {
			x.quote = 0
		}

		return
	}

	if x.naming {
		if character == '>' || isXMLSpace(character) || (character == '/' && len(x.name) > 0) {
			x.naming = false
		} else {
			if len(x.name) < tagNameLimit {
				x.name = append(x.name, character)
			}

			switch {
			case bytes.Equal(x.name, []byte("!--")):
				x.state = stateComment
				x.dashes = 0

				return
			case bytes.Equal(x.name, []byte("![CDATA[")):
				x.state = stateCData
				x.brackets = 0

				return
			}

			return
		}
	}

	switch character {
	case '"', '\'':
		x.quote = character
	case '>':
		x.state = stateText
		x.separator()
	}
}

func (x *xmlText) separator() {
	name := string(x.name)
	closing := strings.HasPrefix(name, "/")
	local := strings.TrimPrefix(name, "/")

	if index := strings.IndexByte(local, ':'); index >= 0 {
		local = local[index+1:]
	}

	switch {
	case closing && (local == "p" || local == "si" || local == "row" || local == "tr"):
		x.out = append(x.out, '\n')
	case !closing && (local == "br" || local == "cr"):
		x.out = append(x.out, '\n')
	case closing && (local == "c" || local == "tc"):
		x.out = append(x.out, '\t')
	case !closing && local == "tab":
		x.out = append(x.out, '\t')
	}
}

func (x *xmlText) entityByte(character byte) {
	if character == ';' {
		x.out = appendEntity(x.out, x.entity)
		x.state = stateText

		return
	}

	if len(x.entity) >= entityLimit || character == '<' || character == '&' || isXMLSpace(character) {
		x.out = append(x.out, '&')
		x.out = append(x.out, x.entity...)
		x.state = stateText
		x.text(character)

		return
	}

	x.entity = append(x.entity, character)
}

func (x *xmlText) comment(character byte) {
	switch {
	case character == '-':
		x.dashes++
	case character == '>' && x.dashes >= 2:
		x.state = stateText
	default:
		x.dashes = 0
	}
}

func (x *xmlText) cdata(character byte) {
	switch {
	case character == ']':
		x.brackets++
	case character == '>' && x.brackets >= 2:
		for range x.brackets - 2 {
			x.out = append(x.out, ']')
		}

		x.brackets = 0
		x.state = stateText
	default:
		for range x.brackets {
			x.out = append(x.out, ']')
		}

		x.brackets = 0
		x.out = append(x.out, character)
	}
}

func appendEntity(out, entity []byte) []byte {
	switch string(entity) {
	case "amp":
		return append(out, '&')
	case "lt":
		return append(out, '<')
	case "gt":
		return append(out, '>')
	case "quot":
		return append(out, '"')
	case "apos":
		return append(out, '\'')
	}

	if len(entity) > 1 && entity[0] == '#' {
		base, digits := 10, entity[1:]
		if digits[0] == 'x' || digits[0] == 'X' {
			base, digits = 16, digits[1:]
		}

		if value, err := strconv.ParseUint(string(digits), base, 32); err == nil && utf8.ValidRune(rune(value)) {
			return utf8.AppendRune(out, rune(value))
		}
	}

	out = append(out, '&')
	out = append(out, entity...)

	return append(out, ';')
}

func isXMLSpace(character byte) bool {
	return character == ' ' || character == '\t' || character == '\n' || character == '\r'
}
