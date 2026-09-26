package oracle

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/cgalvisleon/et/jsql"
)

// maxIdentLen is the identifier length limit since Oracle 12.2.
const maxIdentLen = 128

var identifierUnsafe = regexp.MustCompile(`[^A-Za-z0-9_$#]`)

/**
* sanitizeIdent: Strips anything that isn't a letter, digit, _, $ or #, so a name can never
* break out of the identifier position it's placed in.
* @param s string
* @return string
**/
func sanitizeIdent(s string) string {
	return identifierUnsafe.ReplaceAllString(s, "")
}

/**
* oraIdent: Returns name as a quoted identifier ("name"), keeping its case.
* Quoted identifiers allow names such as _source or _idx, which are invalid unquoted in Oracle.
* @param name string
* @return string
**/
func oraIdent(name string) string {
	return fmt.Sprintf(`"%s"`, sanitizeIdent(name))
}

/**
* oraObjectName: Returns a quoted name for a constraint or index, shortened with a hash
* when it exceeds the Oracle identifier limit.
* @param parts ...string
* @return string
**/
func oraObjectName(parts ...string) string {
	name := sanitizeIdent(strings.Join(parts, "_"))
	if len(name) > maxIdentLen {
		sum := sha1.Sum([]byte(name))
		hash := hex.EncodeToString(sum[:])[:10]
		name = name[:maxIdentLen-11] + "_" + hash
	}
	return fmt.Sprintf(`"%s"`, name)
}

/**
* oraSchema: Returns the schema as an unquoted identifier (resolved in uppercase by Oracle).
* @param schema string
* @return string
**/
func oraSchema(schema string) string {
	return strings.ToUpper(sanitizeIdent(schema))
}

/**
* oraTableRef: Returns the qualified table reference (SCHEMA."name") of a model origin.
* @param schema, name string
* @return string
**/
func oraTableRef(schema, name string) string {
	if schema == "" {
		return oraIdent(name)
	}
	return fmt.Sprintf("%s.%s", oraSchema(schema), oraIdent(name))
}

/**
* oraFromRef: Returns the qualified table reference for a From.
* @param from *jsql.From
* @return string
**/
func oraFromRef(from *jsql.From) string {
	return oraTableRef(from.Schema, from.Name)
}

/**
* oraAlias: Returns the SQL alias of a From, or empty string when it has none
* (a From built without alias uses the table name as As).
* @param from *jsql.From
* @return string
**/
func oraAlias(from *jsql.From) string {
	if from == nil || from.As == "" || from.As == from.Table || strings.Contains(from.As, ".") {
		return ""
	}
	return sanitizeIdent(from.As)
}

/**
* oraColumnRef: Returns alias."column", or "column" when alias is empty.
* @param alias, column string
* @return string
**/
func oraColumnRef(alias, column string) string {
	if alias == "" {
		return oraIdent(column)
	}
	return fmt.Sprintf("%s.%s", alias, oraIdent(column))
}

/**
* oraJsonPath: Builds a SQL/JSON path literal ('$."a"."b"') from parts. Each key is
* double-quoted with \ and " escaped, and the literal is SQL-escaped.
* @param parts []string
* @return string
**/
func oraJsonPath(parts []string) string {
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
* oraQuoteKey: Quotes a JSON key as an Oracle string literal ('key') for JSON_OBJECT.
* @param key string
* @return string
**/
func oraQuoteKey(key string) string {
	return fmt.Sprintf("'%s'", jsql.EscapeSQLString(key))
}
