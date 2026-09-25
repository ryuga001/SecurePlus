package scanner_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/unicode"

	"dpdp-backend/internal/datadiscovery/scanner"
)

func extract(t *testing.T, processor scanner.Processor, input []byte) (string, error) {
	t.Helper()

	var out bytes.Buffer
	err := processor.Extract(context.Background(), bytes.NewReader(input), &out)

	return out.String(), err
}

func limits(t *testing.T) scanner.Limits {
	t.Helper()

	return scanner.DefaultLimits(t.TempDir(), 64<<20)
}

func zipArchive(t *testing.T, parts map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	for name, content := range parts {
		part, err := writer.Create(name)
		if err != nil {
			t.Fatalf("zip create: %v", err)
		}

		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatalf("zip write: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}

	return buffer.Bytes()
}

func TestTextProcessorPassesThroughUTF8(t *testing.T) {
	text, err := extract(t, scanner.TextProcessor{}, []byte("\xEF\xBB\xBFname,pan\nRavi,ABCDE1234F\n"))
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if text != "name,pan\nRavi,ABCDE1234F\n" {
		t.Fatalf("text = %q", text)
	}
}

func TestTextProcessorDecodesUTF16(t *testing.T) {
	encoded, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes([]byte("Aadhaar 1234"))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	text, err := extract(t, scanner.TextProcessor{}, encoded)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if text != "Aadhaar 1234" {
		t.Fatalf("text = %q", text)
	}
}

func TestTextProcessorHandlesEmptyAndTinyFiles(t *testing.T) {
	for _, input := range []string{"", "a", "ab"} {
		text, err := extract(t, scanner.TextProcessor{}, []byte(input))
		if err != nil || text != input {
			t.Fatalf("input %q: text %q err %v", input, text, err)
		}
	}
}

func TestOfficeProcessorExtractsDocxText(t *testing.T) {
	document := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		`<w:p><w:r><w:t>Customer PAN: ABCDE</w:t></w:r><w:r><w:rPr><w:b/></w:rPr><w:t>1234F</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t xml:space="preserve">Tom &amp; Jerry &lt;ok&gt; &#8377;5</w:t></w:r><w:r><w:tab/><w:t>tabbed</w:t></w:r></w:p>` +
		`<!-- ignored > comment --><w:p><w:r><w:t><![CDATA[raw <text>]]></w:t></w:r></w:p>` +
		`</w:body></w:document>`

	archive := zipArchive(t, map[string]string{
		"[Content_Types].xml":  `<Types/>`,
		"word/document.xml":    document,
		"word/styles.xml":      `<w:styles><w:t>STYLE-SHOULD-NOT-APPEAR</w:t></w:styles>`,
		"word/header1.xml":     `<w:hdr><w:p><w:r><w:t>Header note</w:t></w:r></w:p></w:hdr>`,
		"word/media/image.xml": `<x>MEDIA-SHOULD-NOT-APPEAR</x>`,
	})

	text, err := extract(t, scanner.OfficeProcessor{Limits: limits(t)}, archive)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	for _, expected := range []string{
		"Customer PAN: ABCDE1234F\n",
		"Tom & Jerry <ok> ₹5\ttabbed\n",
		"raw <text>\n",
		"Header note",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing %q in %q", expected, text)
		}
	}

	if strings.Contains(text, "SHOULD-NOT-APPEAR") || strings.Contains(text, "ignored") {
		t.Fatalf("unexpected content in %q", text)
	}
}

func TestOfficeProcessorExtractsXlsxAndPptx(t *testing.T) {
	workbook := zipArchive(t, map[string]string{
		"xl/sharedStrings.xml":     `<sst><si><t>Aadhaar</t></si><si><t>Mobile</t></si></sst>`,
		"xl/worksheets/sheet1.xml": `<worksheet><sheetData><row><c t="s"><v>0</v></c><c><v>123456789012</v></c></row></sheetData></worksheet>`,
	})

	text, err := extract(t, scanner.OfficeProcessor{Limits: limits(t)}, workbook)
	if err != nil {
		t.Fatalf("xlsx extract: %v", err)
	}

	if !strings.Contains(text, "Aadhaar\nMobile\n") || !strings.Contains(text, "0\t123456789012\t\n") {
		t.Fatalf("xlsx text = %q", text)
	}

	deck := zipArchive(t, map[string]string{
		"ppt/slides/slide1.xml":           `<p:sld><a:p><a:r><a:t>Salary slip</a:t></a:r></a:p></p:sld>`,
		"ppt/notesSlides/notesSlide1.xml": `<p:notes><a:p><a:r><a:t>Speaker note</a:t></a:r></a:p></p:notes>`,
		"ppt/slides/_rels/slide1.xml":     `<Relationships>REL-SHOULD-NOT-APPEAR</Relationships>`,
	})

	text, err = extract(t, scanner.OfficeProcessor{Limits: limits(t)}, deck)
	if err != nil {
		t.Fatalf("pptx extract: %v", err)
	}

	if !strings.Contains(text, "Salary slip\n") || !strings.Contains(text, "Speaker note\n") || strings.Contains(text, "REL-") {
		t.Fatalf("pptx text = %q", text)
	}
}

func TestOfficeProcessorEnforcesExtractionBudget(t *testing.T) {
	bomb := zipArchive(t, map[string]string{
		"word/document.xml": "<w:t>" + strings.Repeat("A", 4<<20) + "</w:t>",
	})

	processor := scanner.OfficeProcessor{Limits: limits(t)}
	processor.Limits.MaxExtractedBytes = 1 << 20

	if _, err := extract(t, processor, bomb); !errors.Is(err, scanner.ErrLimitExceeded) {
		t.Fatalf("error = %v", err)
	}
}

func TestOfficeProcessorRejectsOversizedDirectory(t *testing.T) {
	parts := make(map[string]string, 200)
	for index := range 200 {
		parts[fmt.Sprintf("word/media/part-%04d.bin", index)] = "x"
	}

	processor := scanner.OfficeProcessor{Limits: limits(t)}
	processor.Limits.MaxZipDirectory = 1024

	if _, err := extract(t, processor, zipArchive(t, parts)); !errors.Is(err, scanner.ErrLimitExceeded) {
		t.Fatalf("error = %v", err)
	}
}

func TestOfficeProcessorRejectsOversizedSpool(t *testing.T) {
	processor := scanner.OfficeProcessor{Limits: limits(t)}
	processor.Limits.MaxSpoolBytes = 1024

	if _, err := extract(t, processor, bytes.Repeat([]byte("x"), 4096)); !errors.Is(err, scanner.ErrFileTooLarge) {
		t.Fatalf("error = %v", err)
	}

	assertNoSpoolFiles(t, processor.Limits.SpoolDir)
}

func TestOfficeProcessorRejectsCorruptArchive(t *testing.T) {
	processor := scanner.OfficeProcessor{Limits: limits(t)}

	if _, err := extract(t, processor, []byte("definitely not a zip archive")); !errors.Is(err, scanner.ErrParse) {
		t.Fatalf("error = %v", err)
	}

	assertNoSpoolFiles(t, processor.Limits.SpoolDir)
}

func TestPDFProcessorExtractsPageText(t *testing.T) {
	processor := scanner.PDFProcessor{Limits: limits(t)}

	text, err := extract(t, processor, minimalPDF("Hello PAN ABCDE1234F"))
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if !strings.Contains(strings.ReplaceAll(text, " ", ""), "HelloPANABCDE1234F") {
		t.Fatalf("text = %q", text)
	}

	assertNoSpoolFiles(t, processor.Limits.SpoolDir)
}

func TestPDFProcessorRejectsMalformedInput(t *testing.T) {
	processor := scanner.PDFProcessor{Limits: limits(t)}

	if _, err := extract(t, processor, []byte("%PDF-1.4\nthis is not really a pdf")); !errors.Is(err, scanner.ErrParse) {
		t.Fatalf("error = %v", err)
	}
}

func TestPDFProcessorEnforcesSizeCap(t *testing.T) {
	processor := scanner.PDFProcessor{Limits: limits(t)}
	processor.Limits.MaxPDFBytes = 128

	if _, err := extract(t, processor, minimalPDF(strings.Repeat("x", 512))); !errors.Is(err, scanner.ErrFileTooLarge) {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveExtension(t *testing.T) {
	cases := []struct {
		name, mime, want string
	}{
		{"report.PDF", "", "pdf"},
		{"folder/data.csv", "application/octet-stream", "csv"},
		{"archive.tar.gz", "", "gz"},
		{"README", "text/plain; charset=utf-8", "txt"},
		{"blob", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx"},
		{"blob", "image/png", ""},
		{"blob", "", ""},
		{"blob", "not a mime type", ""},
	}

	for _, tc := range cases {
		if got := scanner.ResolveExtension(tc.name, tc.mime); got != tc.want {
			t.Fatalf("ResolveExtension(%q, %q) = %q, want %q", tc.name, tc.mime, got, tc.want)
		}
	}
}

func TestDefaultProcessorsCoverSupportedFormats(t *testing.T) {
	processors := scanner.DefaultProcessors(limits(t))

	for _, extension := range []string{"txt", "csv", "json", "xml", "js", "docx", "xlsx", "pptx", "pdf"} {
		if _, ok := processors.For(extension); !ok {
			t.Fatalf("missing processor for %s", extension)
		}
	}

	for _, extension := range []string{"doc", "xls", "ppt", "zip", "png", "exe"} {
		if _, ok := processors.For(extension); ok {
			t.Fatalf("unexpected processor for %s", extension)
		}
	}
}

func TestRemoveStaleSpools(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"dpdp-scan-1", "dpdp-scan-2", "keep.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	removed, err := scanner.RemoveStaleSpools(dir)
	if err != nil || removed != 2 {
		t.Fatalf("removed = %d err = %v", removed, err)
	}

	if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err != nil {
		t.Fatalf("unrelated file removed: %v", err)
	}
}

func assertNoSpoolFiles(t *testing.T, dir string) {
	t.Helper()

	matches, _ := filepath.Glob(filepath.Join(dir, scanner.SpoolPattern))
	if len(matches) > 0 {
		t.Fatalf("spool files left behind: %v", matches)
	}
}

func minimalPDF(text string) []byte {
	content := fmt.Sprintf("BT /F1 24 Tf 72 700 Td (%s) Tj ET", text)

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}

	var buffer bytes.Buffer
	buffer.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(objects))
	for index, object := range objects {
		offsets[index] = buffer.Len()
		fmt.Fprintf(&buffer, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}

	xref := buffer.Len()
	fmt.Fprintf(&buffer, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)

	for _, offset := range offsets {
		fmt.Fprintf(&buffer, "%010d 00000 n \n", offset)
	}

	fmt.Fprintf(&buffer, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)

	return buffer.Bytes()
}
