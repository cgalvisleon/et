package postgres

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

// identifierUnsafe matches any character not allowed in a bare SQL
// identifier segment (letters, digits, underscore). Used to strip anything
// an attacker could use to break out of the generated SQL when a field name
// coming from outside the model's column whitelist is used to build an
// expression (e.g. quotes, semicolons, comment markers, whitespace).
var identifierUnsafe = regexp.MustCompile(`[^A-Za-z0-9_]`)
var qualifiedIdentifierUnsafe = regexp.MustCompile(`[^A-Za-z0-9_.]`)

/**
* sanitizeIdent: Strips anything that isn't a letter, digit or underscore,
* so a value can never break out of the identifier position it's placed in.
* @param s string
* @return string
**/
func sanitizeIdent(s string) string {
	return identifierUnsafe.ReplaceAllString(s, "")
}

/**
* sanitizeQualifiedIdent: Like sanitizeIdent but also allows "." for an
* already schema/table-qualified identifier (e.g. "table.column").
* @param s string
* @return string
**/
func sanitizeQualifiedIdent(s string) string {
	return qualifiedIdentifierUnsafe.ReplaceAllString(s, "")
}

/**
* pgQuoteKey: Quotes a JSON key as a PostgreSQL string literal ('key'), escaping single quotes.
* Used for ->'key', ->>'key' and the keys of jsonb_build_object.
* @param key string
* @return string
**/
func pgQuoteKey(key string) string {
	return fmt.Sprintf("'%s'", jsql.EscapeSQLString(key))
}

/**
* pgTextArray: Builds a PostgreSQL text[] literal ('{"a","b"}') from parts.
* Each element is double-quoted with \ and " escaped, so keys containing commas,
* braces, quotes or spaces keep their exact value; the whole literal is SQL-escaped.
* Used for jsonb paths (jsonb_set, #>) and for key lists (jsonb - text[]).
* @param parts []string
* @return string
**/
func pgTextArray(parts []string) string {
	elems := make([]string, len(parts))
	for i, p := range parts {
		p = strings.ReplaceAll(p, `\`, `\\`)
		p = strings.ReplaceAll(p, `"`, `\"`)
		elems[i] = `"` + p + `"`
	}
	return fmt.Sprintf("'{%s}'", jsql.EscapeSQLString(strings.Join(elems, ",")))
}

/**
* pgAttribCast: Returns the PostgreSQL cast type for JSONB text extraction
* when the ATTRIB TypeData requires a non-text comparison.
* Returns empty string for text types (no cast needed).
* @param tp et.TypeData
* @return string
**/
func pgAttribCast(tp et.TypeData) string {
	switch tp {
	case et.INT:
		return "BIGINT"
	case et.FLOAT:
		return "DOUBLE PRECISION"
	case et.BOOL:
		return "BOOLEAN"
	case et.DATETIME:
		return "TIMESTAMP"
	default:
		return ""
	}
}

/**
* pgValueCast: Returns the cast needed to compare an untyped JSONB text extraction
* (->>) against val: numbers compare as NUMERIC, booleans as BOOLEAN, times as TIMESTAMP.
* Slices use their first element (IN lists). Returns empty string for text values.
* @param val any
* @return string
**/
func pgValueCast(val any) string {
	switch v := val.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return "NUMERIC"
	case bool:
		return "BOOLEAN"
	case time.Time, *time.Time:
		return "TIMESTAMP"
	case et.BetweenValue:
		return pgValueCast(v.Min)
	case []any:
		if len(v) > 0 {
			return pgValueCast(v[0])
		}
	case []int, []int64, []float64:
		return "NUMERIC"
	}
	return ""
}
