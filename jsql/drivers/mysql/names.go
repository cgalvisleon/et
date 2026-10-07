package mysql

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/cgalvisleon/et/jsql"
)

// maxIdentLen is the MySQL identifier length limit.
const maxIdentLen = 64

var identifierUnsafe = regexp.MustCompile(`[^A-Za-z0-9_$]`)

/**
* sanitizeIdent: Strips anything that isn't a letter, digit, _ or $, so a name can never break out of
* the identifier position it's placed in.
* @param s string
* @return string
**/
func sanitizeIdent(s string) string {
	return identifierUnsafe.ReplaceAllString(s, "")
}

/**
* myIdent: Returns name quoted with backticks.
* @param name string
* @return string
**/
func myIdent(name string) string {
	return "`" + sanitizeIdent(name) + "`"
}

/**
* myObjectName: Returns a quoted name for an index or constraint, shortened with a hash when it exceeds
* the identifier limit.
* @param parts ...string
* @return string
**/
func myObjectName(parts ...string) string {
	name := sanitizeIdent(strings.Join(parts, "_"))
	if len(name) > maxIdentLen {
		sum := sha1.Sum([]byte(name))
		name = name[:maxIdentLen-11] + "_" + hex.EncodeToString(sum[:])[:10]
	}
	return "`" + name + "`"
}

/**
* myFromRef: Returns the quoted table reference of a From: its Table (schema_name).
* @param from *jsql.From
* @return string
**/
func myFromRef(from *jsql.From) string {
	return myIdent(from.Table)
}

/**
* myAlias: Returns the SQL alias of a From, or "" when it has none (its As is the table name).
* @param from *jsql.From
* @return string
**/
func myAlias(from *jsql.From) string {
	if from == nil || from.As == "" || from.As == from.Table || strings.Contains(from.As, ".") {
		return ""
	}
	return sanitizeIdent(from.As)
}

/**
* myColumnRef: Returns alias.`column`, or `column` without alias.
* @param alias, column string
* @return string
**/
func myColumnRef(alias, column string) string {
	if alias == "" {
		return myIdent(column)
	}
	return fmt.Sprintf("%s.%s", alias, myIdent(column))
}

/**
* myJsonPath: Builds a JSON path literal ('$."a"."b"'); keys are double-quoted with \ and " escaped.
* @param parts []string
* @return string
**/
func myJsonPath(parts []string) string {
	var sb strings.Builder
	sb.WriteString("$")
	for _, p := range parts {
		p = strings.ReplaceAll(p, `\`, `\\`)
		p = strings.ReplaceAll(p, `"`, `\"`)
		sb.WriteString(`."` + p + `"`)
	}
	return myQuoteText(sb.String())
}

/**
* myQuoteKey: Quotes a JSON key as a string literal for JSON_OBJECT.
* @param key string
* @return string
**/
func myQuoteKey(key string) string {
	return myQuoteText(key)
}
