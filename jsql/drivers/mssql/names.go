package mssql

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/cgalvisleon/et/jsql"
)

// maxIdentLen is the SQL Server identifier length limit.
const maxIdentLen = 128

var identifierUnsafe = regexp.MustCompile(`[^A-Za-z0-9_$#@]`)

/**
* sanitizeIdent: Strips anything that isn't a letter, digit, _, $, # or @.
* @param s string
* @return string
**/
func sanitizeIdent(s string) string {
	return identifierUnsafe.ReplaceAllString(s, "")
}

/**
* msIdent: Returns name quoted with brackets.
* @param name string
* @return string
**/
func msIdent(name string) string {
	return "[" + sanitizeIdent(name) + "]"
}

/**
* msObjectName: Returns a quoted name for an index or constraint, shortened with a hash when too long.
* @param parts ...string
* @return string
**/
func msObjectName(parts ...string) string {
	name := sanitizeIdent(strings.Join(parts, "_"))
	if len(name) > maxIdentLen {
		sum := sha1.Sum([]byte(name))
		name = name[:maxIdentLen-11] + "_" + hex.EncodeToString(sum[:])[:10]
	}
	return "[" + name + "]"
}

/**
* msTableRef: Returns [schema].[name] ([dbo] when the model has no schema).
* @param schema, name string
* @return string
**/
func msTableRef(schema, name string) string {
	if schema == "" {
		schema = "dbo"
	}
	return fmt.Sprintf("%s.%s", msIdent(schema), msIdent(name))
}

/**
* msFromRef: Returns the table reference of a From.
* @param from *jsql.From
* @return string
**/
func msFromRef(from *jsql.From) string {
	return msTableRef(from.Schema, from.Name)
}

/**
* msAlias: Returns the SQL alias of a From, or "" when it has none (its As is the table name).
* @param from *jsql.From
* @return string
**/
func msAlias(from *jsql.From) string {
	if from == nil || from.As == "" || from.As == from.Table || strings.Contains(from.As, ".") {
		return ""
	}
	return sanitizeIdent(from.As)
}

/**
* msColumnRef: Returns alias.[column], or [column] without alias.
* @param alias, column string
* @return string
**/
func msColumnRef(alias, column string) string {
	if alias == "" {
		return msIdent(column)
	}
	return fmt.Sprintf("%s.%s", alias, msIdent(column))
}

/**
* msJsonPathText: Returns the SQL/JSON path '$."a"."b"' (without SQL quoting); keys are double-quoted with
* \ and " escaped.
* @param parts []string
* @return string
**/
func msJsonPathText(parts []string) string {
	var sb strings.Builder
	sb.WriteString("$")
	for _, p := range parts {
		p = strings.ReplaceAll(p, `\`, `\\`)
		p = strings.ReplaceAll(p, `"`, `\"`)
		sb.WriteString(`."` + p + `"`)
	}
	return sb.String()
}

/**
* msJsonPath: Returns a JSON path as a string literal.
* @param parts []string
* @return string
**/
func msJsonPath(parts []string) string {
	return msQuoteText(msJsonPathText(parts))
}

/**
* msKeyPrefix: Returns the literal N'"key":' that starts a pair of a JSON object built as text.
* @param key string
* @return string
**/
func msKeyPrefix(key string) string {
	bt, _ := json.Marshal(key)
	return msQuoteText(string(bt) + ":")
}
