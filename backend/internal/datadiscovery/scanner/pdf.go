package scanner

import (
	"context"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

type PDFProcessor struct {
	Limits Limits
}

func (p PDFProcessor) Extract(ctx context.Context, r io.Reader, w io.Writer) (err error) {
	limit := p.Limits.MaxPDFBytes
	if p.Limits.MaxSpoolBytes > 0 && p.Limits.MaxSpoolBytes < limit {
		limit = p.Limits.MaxSpoolBytes
	}

	file, err := spool(ctx, r, p.Limits.SpoolDir, limit)
	if err != nil {
		return err
	}
	defer discard(file)

	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w: %v", ErrParse, recovered)
		}
	}()

	reader, err := pdf.NewReader(file, file.size)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrParse, err)
	}

	pages := reader.NumPage()

	for number := 1; number <= pages; number++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		page := reader.Page(number)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			return fmt.Errorf("%w: page %d: %v", ErrParse, number, err)
		}

		if _, err := io.WriteString(w, text); err != nil {
			return err
		}

		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}

	return nil
}
