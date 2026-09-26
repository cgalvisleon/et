package sqlite

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

// qualifiedIdentifierUnsafe matches any character not allowed in a (possibly
// table-qualified) identifier, so a field name coming from outside the model can
// never break out of the identifier position it is placed in.
var qualifiedIdentifierUnsafe = regexp.MustCompile(`[^A-Za-z0-9_.]`)

/**
* sanitizeQualifiedIdent: Strips anything that isn't a letter, digit, "_" or ".".
* @param s string
* @return string
**/
func sanitizeQualifiedIdent(s string) string {
	return qualifiedIdentifierUnsafe.ReplaceAllString(s, "")
}

/**
* sqliteAttribCast: Returns the SQLite CAST type for JSON text extraction when the
* ATTRIB TypeData requires a non-text comparison. Returns empty string for text
* types (no cast needed).
* @param tp et.TypeData
* @return string
**/
func sqliteAttribCast(tp et.TypeData) string {
	switch tp {
	case et.INT:
		return "INTEGER"
	case et.FLOAT:
		return "REAL"
	case et.BOOL:
		return "INTEGER"
	case et.DATETIME:
		return "TEXT"
	default:
		return ""
	}
}

/**
* sqliteQuoteKey: Quotes a JSON key as a SQLite string literal ('key') for json_object.
* @param key string
* @return string
**/
func sqliteQuoteKey(key string) string {
	return fmt.Sprintf("'%s'", jsql.EscapeSQLString(key))
}

/**
* sqlitePath: Builds a JSON path literal ('$."a"."b"') from parts. Each key is double-quoted
* with \ and " escaped, so keys with dots, quotes or spaces keep their exact value.
* @param parts []string
* @return string
**/
func sqlitePath(parts []string) string {
	var sb strings.Builder
	sb.WriteString("$")
	for _, p := range parts {
		p = strings.ReplaceAll(p, `\`, `\\`)
		p = strings.ReplaceAll(p, `"`, `\"`)
		sb.WriteString(`."` + p + `"`)
	}
	return fmt.Sprintf("'%s'", jsql.EscapeSQLString(sb.String()))
}

/**
* sqliteColumnJson: Returns a column as a JSON value for json_object/json_set: JSON columns are
* parsed with json(), booleans become true/false; other columns keep their SQL value.
* @param ref string, tp et.TypeData
* @return string
**/
func sqliteColumnJson(ref string, tp et.TypeData) string {
	switch tp {
	case et.BOOL:
		return fmt.Sprintf("json(CASE %s WHEN 1 THEN 'true' WHEN 0 THEN 'false' END)", ref)
	case et.JSON, et.ARRAY, et.ARRAY_JSON, et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME, et.VAL_BETWEEN:
		return fmt.Sprintf("json(%s)", ref)
	default:
		return ref
	}
}

/**
* sqliteSourceExpr: Returns the SourceField as a JSON object without the hidden keys;
* a NULL source counts as '{}'.
* @param source string, hiddens []string
* @return string
**/
func sqliteSourceExpr(source string, hiddens []string) string {
	expr := fmt.Sprintf("COALESCE(%s, '{}')", source)
	if len(hiddens) == 0 {
		return expr
	}
	paths := make([]string, len(hiddens))
	for i, h := range hiddens {
		paths[i] = sqlitePath([]string{h})
	}
	return fmt.Sprintf("json_remove(%s, %s)", expr, strings.Join(paths, ", "))
}

/**
* sqliteSetObject: Builds json_set(base, path, value, ...) from pairs ("path, value"), nesting
* calls in chunks of 50 pairs to stay under SQLite's function argument limit.
* json_set keeps null values and creates missing parent objects.
* @param base string, pairs []string
* @return string
**/
func sqliteSetObject(base string, pairs []string) string {
	const maxPairs = 50
	expr := base
	for i := 0; i < len(pairs); i += maxPairs {
		end := min(i+maxPairs, len(pairs))
		expr = fmt.Sprintf("json_set(%s,\n%s\n)", expr, strings.Join(pairs[i:end], ",\n"))
	}
	return expr
}
