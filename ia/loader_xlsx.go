package ia

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

/**
* extractXlsxText: Renders every sheet of an Excel workbook as text, one
* "Sheet: <name>" header per sheet followed by its rows joined with commas.
* @param data []byte
* @return string, error
**/
func extractXlsxText(data []byte) (string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer f.Close()

	var sb strings.Builder
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil {
			continue
		}
		fmt.Fprintf(&sb, "Sheet: %s\n", name)
		for _, row := range rows {
			sb.WriteString(strings.Join(row, ", "))
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}
