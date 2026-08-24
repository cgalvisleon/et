package ia

import (
	"fmt"
	"sort"
	"strings"

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

	var sb strings.Builder
	for _, row := range items.Result {
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

	return sb.String(), nil
}
