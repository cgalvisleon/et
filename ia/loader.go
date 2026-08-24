package ia

import (
	"fmt"
	"strings"
)

const (
	SourcePDF  = "pdf"
	SourceDOCX = "docx"
	SourceXLSX = "xlsx"
	SourceCSV  = "csv"
	SourceTXT  = "txt"
	SourceMD   = "md"
	SourceSQL  = "sql"
)

/**
* sourceFromFilename: Infers the Source constant from a filename's extension.
* @param filename string
* @return string, error
**/
func sourceFromFilename(filename string) (string, error) {
	ext := strings.ToLower(filename)
	if idx := strings.LastIndex(ext, "."); idx != -1 {
		ext = ext[idx+1:]
	}

	switch ext {
	case "pdf":
		return SourcePDF, nil
	case "docx":
		return SourceDOCX, nil
	case "xlsx":
		return SourceXLSX, nil
	case "csv":
		return SourceCSV, nil
	case "txt":
		return SourceTXT, nil
	case "md", "markdown":
		return SourceMD, nil
	default:
		return "", fmt.Errorf(MSG_UNSUPPORTED_SOURCE, ext)
	}
}

/**
* extractText: Extracts plain text from data according to source, dispatching
* to the format-specific loader (csv, xlsx, docx, pdf) or returning the raw
* bytes as text (txt, md).
* @param source string, data []byte
* @return string, error
**/
func extractText(source string, data []byte) (string, error) {
	switch source {
	case SourceTXT, SourceMD:
		return string(data), nil
	case SourceCSV:
		return extractCsvText(data)
	case SourceXLSX:
		return extractXlsxText(data)
	case SourceDOCX:
		return extractDocxText(data)
	case SourcePDF:
		return extractPdfText(data)
	default:
		return "", fmt.Errorf(MSG_UNSUPPORTED_SOURCE, source)
	}
}
