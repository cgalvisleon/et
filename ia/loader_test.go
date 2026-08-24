package ia

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestExtractXlsxText(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	f.SetCellValue("Sheet1", "A1", "name")
	f.SetCellValue("Sheet1", "B1", "amount")
	f.SetCellValue("Sheet1", "A2", "widget")
	f.SetCellValue("Sheet1", "B2", 42)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write xlsx: %v", err)
	}

	text, err := extractXlsxText(buf.Bytes())
	if err != nil {
		t.Fatalf("extractXlsxText: %v", err)
	}
	if !strings.Contains(text, "Sheet1") || !strings.Contains(text, "widget") {
		t.Fatalf("unexpected xlsx text: %q", text)
	}
}

func buildMinimalDocx(t *testing.T, paragraphs ...string) []byte {
	t.Helper()

	var body strings.Builder
	for _, p := range paragraphs {
		body.WriteString(`<w:p><w:r><w:t>`)
		body.WriteString(p)
		body.WriteString(`</w:t></w:r></w:p>`)
	}

	doc := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>` + body.String() + `</w:body>
</w:document>`

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write([]byte(doc)); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}

	return buf.Bytes()
}

func TestExtractDocxText(t *testing.T) {
	data := buildMinimalDocx(t, "Hola mundo", "Segundo parrafo")

	text, err := extractDocxText(data)
	if err != nil {
		t.Fatalf("extractDocxText: %v", err)
	}
	if !strings.Contains(text, "Hola mundo") || !strings.Contains(text, "Segundo parrafo") {
		t.Fatalf("unexpected docx text: %q", text)
	}
}

func TestSourceFromFilename(t *testing.T) {
	cases := map[string]string{
		"report.pdf": SourcePDF,
		"doc.DOCX":   SourceDOCX,
		"sheet.xlsx": SourceXLSX,
		"data.csv":   SourceCSV,
		"notes.txt":  SourceTXT,
		"readme.md":  SourceMD,
	}
	for filename, want := range cases {
		got, err := sourceFromFilename(filename)
		if err != nil {
			t.Fatalf("sourceFromFilename(%q): %v", filename, err)
		}
		if got != want {
			t.Fatalf("sourceFromFilename(%q) = %q, want %q", filename, got, want)
		}
	}

	if _, err := sourceFromFilename("archive.zip"); err == nil {
		t.Fatal("expected an error for an unsupported extension")
	}
}
