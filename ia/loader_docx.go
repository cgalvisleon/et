package ia

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

/**
* extractDocxText: Extracts the plain text of a .docx document by reading
* word/document.xml out of its zip container and concatenating the text
* runs (<w:t>), inserting a newline at each paragraph boundary (<w:p>).
* @param data []byte
* @return string, error
**/
func extractDocxText(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	var doc *zip.File
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			doc = f
			break
		}
	}
	if doc == nil {
		return "", nil
	}

	rc, err := doc.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	var sb strings.Builder
	decoder := xml.NewDecoder(rc)
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				var text string
				if err := decoder.DecodeElement(&text, &t); err != nil {
					return "", err
				}
				sb.WriteString(text)
			}
		case xml.EndElement:
			if t.Name.Local == "p" {
				sb.WriteString("\n")
			}
		}
	}

	return sb.String(), nil
}
