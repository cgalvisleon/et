package postgres

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
)

/**
* pgType: Maps an et.TypeData to the corresponding PostgreSQL column type.
* @param tp et.TypeData
* @return string
**/
func pgType(tp et.TypeData) string {
	switch tp {
	case et.ANY:
		return "TEXT"
	case et.BYTE:
		return "BYTEA"
	case et.KEY:
		return "VARCHAR(80)"
	case et.TEXT:
		return "VARCHAR(255)"
	case et.MEMO:
		return "TEXT"
	case et.INT:
		return "BIGINT"
	case et.FLOAT:
		return "DOUBLE PRECISION"
	case et.BOOL:
		return "BOOLEAN"
	case et.DATETIME:
		return "TIMESTAMP"
	case et.JSON:
		return "JSONB"
	case et.ARRAY:
		return "JSONB"
	case et.ARRAY_JSON:
		return "JSONB"
	case et.ARRAY_STRING:
		return "TEXT[]"
	case et.ARRAY_INT:
		return "BIGINT[]"
	case et.ARRAY_FLOAT:
		return "DOUBLE PRECISION[]"
	case et.ARRAY_BOOL:
		return "BOOLEAN[]"
	case et.ARRAY_DATETIME:
		return "TIMESTAMP[]"
	case et.VAL_BETWEEN:
		return "JSONB"
	default: // ANY
		return "TEXT"
	}
}

/**
* pgDefault: Returns the SQL DEFAULT expression for a given TypeData and value.
* @param tp et.TypeData, val any
* @return string
**/
func pgDefault(tp et.TypeData, val any) string {
	if val == nil || val == "" {
		return "NULL"
	}
	switch tp {
	case et.INT, et.FLOAT:
		return fmt.Sprintf("%v", val)
	case et.BOOL:
		return fmt.Sprintf("%v", val)
	case et.JSON, et.ARRAY, et.ARRAY_JSON, et.VAL_BETWEEN:
		return pgQuoteJson(val)
	case et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME:
		return "'{}'"
	case et.DATETIME:
		return "NOW()"
	default:
		return pgQuoteString(val)
	}
}
