package mysql

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/jsql"
)

/**
* myType: Maps a jsql TypeData to the corresponding MySQL column type.
* @param tp jsql.TypeData
* @return string
**/
func myType(tp jsql.TypeData) string {
	switch tp {
	case jsql.INT:
		return "BIGINT"
	case jsql.FLOAT:
		return "DOUBLE"
	case jsql.KEY:
		return "VARCHAR(80)"
	case jsql.TEXT:
		return "VARCHAR(255)"
	case jsql.MEMO:
		return "TEXT"
	case jsql.JSON:
		return "JSON"
	case jsql.DATETIME:
		return "DATETIME(6)"
	case jsql.BOOLEAN:
		return "TINYINT(1)"
	case jsql.BYTES:
		return "BLOB"
	case jsql.GEOMETRY:
		return "JSON"
	case jsql.EMBEDDING:
		return "LONGTEXT"
	default:
		return "VARCHAR(255)"
	}
}

/**
* myDefault: Returns the SQL DEFAULT expression for a given TypeData and value.
* JSON columns require expression syntax (MySQL 8.0.13+): DEFAULT (JSON_OBJECT()).
* @param tp jsql.TypeData
* @param val any
* @return string
**/
func myDefault(tp jsql.TypeData, val any) string {
	if val == nil || val == "" {
		return "NULL"
	}
	switch tp {
	case jsql.INT, jsql.FLOAT:
		return fmt.Sprintf("%v", val)
	case jsql.BOOLEAN:
		return fmt.Sprintf("%v", val)
	case jsql.JSON:
		return fmt.Sprintf("(JSON_OBJECT())")
	case jsql.DATETIME:
		return "CURRENT_TIMESTAMP(6)"
	default:
		return fmt.Sprintf("'%v'", val)
	}
}

/**
* myJsonPath: Converts a '->' separated field path into a MySQL JSON path literal ('$.a.b.c').
* @param field string
* @return string
**/
func myJsonPath(field string) string {
	parts := strings.Split(field, "->")
	return "'$." + strings.Join(parts, ".") + "'"
}

/**
* myJsonExtract: Builds a MySQL JSON extraction expression for an ATTRIB field.
* Uses JSON_UNQUOTE(JSON_EXTRACT(...)) for text; CAST(JSON_EXTRACT(...) AS TYPE) for numeric/bool.
* @param col string, field string, tp jsql.TypeData
* @return string
**/
func myJsonExtract(col, field string, tp jsql.TypeData) string {
	path := myJsonPath(field)
	base := fmt.Sprintf("JSON_EXTRACT(%s, %s)", col, path)
	switch tp {
	case jsql.INT:
		return fmt.Sprintf("CAST(%s AS SIGNED)", base)
	case jsql.FLOAT:
		return fmt.Sprintf("CAST(%s AS DECIMAL(20,6))", base)
	case jsql.BOOLEAN:
		return fmt.Sprintf("CAST(%s AS UNSIGNED)", base)
	case jsql.DATETIME:
		return fmt.Sprintf("CAST(JSON_UNQUOTE(%s) AS DATETIME)", base)
	default:
		return fmt.Sprintf("JSON_UNQUOTE(%s)", base)
	}
}
