package ia

import (
	"bytes"
	"io"

	"github.com/ledongthuc/pdf"
)

/**
* extractPdfText: Extracts the plain text of a PDF document.
* @param data []byte
* @return string, error
**/
func extractPdfText(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	r, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}

	text, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	return string(text), nil
}
