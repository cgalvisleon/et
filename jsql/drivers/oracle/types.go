package oracle

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
)

/**
* oraType: Maps an et.TypeData to the corresponding Oracle column type.
* Booleans are NUMBER(1) and JSON values CLOB (checked with IS JSON in the DDL), which work from 19c.
* @param tp et.TypeData
* @return string
**/
func oraType(tp et.TypeData) string {
	switch tp {
	case et.BYTE:
		return "BLOB"
	case et.KEY:
		return "VARCHAR2(80)"
	case et.TEXT:
		return "VARCHAR2(255)"
	case et.MEMO:
		return "CLOB"
	case et.INT:
		return "NUMBER(19)"
	case et.FLOAT:
		return "NUMBER"
	case et.BOOL:
		return "NUMBER(1)"
	case et.DATETIME:
		return "TIMESTAMP"
	case et.JSON, et.ARRAY, et.ARRAY_JSON, et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME, et.VAL_BETWEEN:
		return "CLOB"
	default: // ANY
		return "VARCHAR2(4000)"
	}
}

/**
* isJsonType: Reports whether values of tp are stored as JSON text.
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
* isLobType: Reports whether a column of tp is a LOB, which Oracle cannot index or compare with =.
* @param tp et.TypeData
* @return bool
**/
func isLobType(tp et.TypeData) bool {
	return tp == et.MEMO || tp == et.BYTE || isJsonType(tp)
}

/**
* oraDefault: Returns the SQL DEFAULT expression for a given TypeData and value.
* Oracle stores '' as NULL, so empty strings default to NULL.
* @param tp et.TypeData, val any
* @return string
**/
func oraDefault(tp et.TypeData, val any) string {
	if val == nil || val == "" {
		return "NULL"
	}
	switch tp {
	case et.INT, et.FLOAT:
		return fmt.Sprintf("%v", val)
	case et.BOOL:
		return oraQuoteBool(val)
	case et.DATETIME:
		return "SYSTIMESTAMP"
	case et.BYTE:
		return "NULL"
	default:
		if isJsonType(tp) {
			return oraQuoteJson(val)
		}
		return oraQuoteString(val)
	}
}
