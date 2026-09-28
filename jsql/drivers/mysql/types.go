package mysql

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
)

/**
* myType: Maps an et.TypeData to the corresponding MySQL column type.
* @param tp et.TypeData
* @return string
**/
func myType(tp et.TypeData) string {
	switch tp {
	case et.BYTE:
		return "LONGBLOB"
	case et.KEY:
		return "VARCHAR(80)"
	case et.TEXT:
		return "VARCHAR(255)"
	case et.MEMO:
		return "LONGTEXT"
	case et.INT:
		return "BIGINT"
	case et.FLOAT:
		return "DOUBLE"
	case et.BOOL:
		return "TINYINT(1)"
	case et.DATETIME:
		return "DATETIME(6)"
	default:
		if isJsonType(tp) {
			return "JSON"
		}
		return "TEXT" // ANY
	}
}

/**
* isJsonType: Reports whether values of tp are stored as JSON.
* @param tp et.TypeData
* @return bool
**/
func isJsonType(tp et.TypeData) bool {
	switch tp {
	case et.JSON, et.ARRAY, et.ARRAY_JSON, et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME, et.VAL_BETWEEN:
		return true
	}
	return false
}

/**
* isBlobType: Reports whether a column of tp cannot be indexed without a prefix length (TEXT, BLOB, JSON).
* @param tp et.TypeData
* @return bool
**/
func isBlobType(tp et.TypeData) bool {
	switch myType(tp) {
	case "LONGBLOB", "LONGTEXT", "JSON", "TEXT":
		return true
	}
	return false
}

/**
* myDefault: Returns the DEFAULT expression for a type and value. TEXT, BLOB and JSON columns only
* accept expression defaults, written in parentheses.
* @param tp et.TypeData, val any
* @return string
**/
func myDefault(tp et.TypeData, val any) string {
	if val == nil || val == "" {
		return "NULL"
	}
	switch tp {
	case et.INT, et.FLOAT:
		return fmt.Sprintf("%v", val)
	case et.BOOL:
		return myQuoteBool(val)
	case et.DATETIME:
		return "CURRENT_TIMESTAMP(6)"
	case et.BYTE:
		return "NULL"
	case et.MEMO:
		return fmt.Sprintf("(%s)", myQuoteString(val))
	default:
		if isJsonType(tp) {
			return fmt.Sprintf("(CAST(%s AS JSON))", myQuoteJson(val))
		}
		if myType(tp) == "TEXT" {
			return fmt.Sprintf("(%s)", myQuoteString(val))
		}
		return myQuoteString(val)
	}
}
