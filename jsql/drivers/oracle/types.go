package oracle

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/jsql"
)

/**
* orType: Maps a jsql TypeData to the corresponding Oracle column type.
* JSON data is stored as CLOB (Oracle 19c lacks a native JSONB type).
* BOOLEAN is represented as NUMBER(1) (0/1) since Oracle has no BOOLEAN column type.
* @param tp jsql.TypeData
* @return string
**/
func orType(tp jsql.TypeData) string {
	switch tp {
	case jsql.INT:
		return "NUMBER(19)"
	case jsql.FLOAT:
		return "BINARY_DOUBLE"
	case jsql.KEY:
		return "VARCHAR2(80)"
	case jsql.TEXT:
		return "VARCHAR2(255)"
	case jsql.MEMO:
		return "CLOB"
	case jsql.JSON:
		return "CLOB"
	case jsql.DATETIME:
		return "TIMESTAMP WITH TIME ZONE"
	case jsql.BOOLEAN:
		return "NUMBER(1)"
	case jsql.BYTES:
		return "BLOB"
	case jsql.GEOMETRY:
		return "CLOB"
	case jsql.EMBEDDING:
		return "CLOB"
	default: // ANY
		return "VARCHAR2(255)"
	}
}

/**
* orDefault: Returns the SQL DEFAULT expression for a given TypeData and value.
* DATETIME uses SYSTIMESTAMP; JSON CLOB defaults to '{}'; booleans map to 0/1.
* @param tp jsql.TypeData
* @param val any
* @return string
**/
func orDefault(tp jsql.TypeData, val any) string {
	if val == nil || val == "" {
		return "NULL"
	}
	switch tp {
	case jsql.INT, jsql.FLOAT:
		return fmt.Sprintf("%v", val)
	case jsql.BOOLEAN:
		b, ok := val.(bool)
		if ok {
			if b {
				return "1"
			}
			return "0"
		}
		return fmt.Sprintf("%v", val)
	case jsql.JSON:
		return "'{}'"
	case jsql.DATETIME:
		return "SYSTIMESTAMP"
	default:
		return fmt.Sprintf("'%v'", val)
	}
}

/**
* orJsonPath: Converts a '->' separated field path into an Oracle JSON path literal ('$.a.b.c').
* @param field string
* @return string
**/
func orJsonPath(field string) string {
	parts := strings.Split(field, "->")
	return "'$." + strings.Join(parts, ".") + "'"
}

/**
* orJsonValue: Builds a JSON_VALUE expression with a RETURNING clause matching the column's TypeData.
* Oracle's JSON_VALUE natively returns typed values via RETURNING, avoiding external CASTs.
* @param col string, field string, tp jsql.TypeData
* @return string
**/
func orJsonValue(col, field string, tp jsql.TypeData) string {
	path := orJsonPath(field)
	switch tp {
	case jsql.INT:
		return fmt.Sprintf("JSON_VALUE(%s, %s RETURNING NUMBER(19))", col, path)
	case jsql.FLOAT:
		return fmt.Sprintf("JSON_VALUE(%s, %s RETURNING BINARY_DOUBLE)", col, path)
	case jsql.BOOLEAN:
		return fmt.Sprintf("JSON_VALUE(%s, %s RETURNING NUMBER(1))", col, path)
	case jsql.DATETIME:
		return fmt.Sprintf("JSON_VALUE(%s, %s RETURNING TIMESTAMP WITH TIME ZONE)", col, path)
	default:
		return fmt.Sprintf("JSON_VALUE(%s, %s)", col, path)
	}
}
