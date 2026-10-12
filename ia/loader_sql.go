package ia

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* extractSqlText: Runs query against db and renders the resulting rows as
* text, one "key: value" pair per line, rows separated by a blank line.
* @param db *jsql.DB, query string
* @return string, error
**/
func extractSqlText(db *jsql.DB, query string) (string, error) {
	items, err := db.Sql(query)
	if err != nil {
		return "", err
	}

	return RowsText(items.Result), nil
}

/**
* RowsText: Renders rows as text, one "key: value" pair per line (keys sorted), rows separated by a blank line.
* @param rows []et.Json
* @return string
**/
func RowsText(rows []et.Json) string {
	var sb strings.Builder
	for _, row := range rows {
		keys := make([]string, 0, len(row))
		for k := range row {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fmt.Fprintf(&sb, "%s: %v\n", k, row[k])
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
