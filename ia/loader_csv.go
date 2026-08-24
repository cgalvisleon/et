package ia

import (
	"strings"

	"github.com/cgalvisleon/et/csv"
)

/**
* extractCsvText: Renders CSV rows as newline-separated, comma-joined text lines.
* @param data []byte
* @return string, error
**/
func extractCsvText(data []byte) (string, error) {
	reader, err := csv.ReadCsv(data, 0)
	if err != nil {
		return "", err
	}

	rows, err := reader.GetRows()
	if err != nil {
		return "", err
	}

	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, strings.Join(row, ", "))
	}

	return strings.Join(lines, "\n"), nil
}
