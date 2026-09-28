package mssql

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
)

/**
* msType: Maps an et.TypeData to the corresponding SQL Server column type.
* @param tp et.TypeData
* @return string
**/
func msType(tp et.TypeData) string {
	switch tp {
	case et.BYTE:
		return "VARBINARY(MAX)"
	case et.KEY:
		return "NVARCHAR(80)"
	case et.TEXT:
		return "NVARCHAR(255)"
	case et.MEMO:
		return "NVARCHAR(MAX)"
	case et.INT:
		return "BIGINT"
	case et.FLOAT:
		return "FLOAT"
	case et.BOOL:
		return "BIT"
	case et.DATETIME:
		return "DATETIME2"
	default:
		if isJsonType(tp) {
			return "NVARCHAR(MAX)"
		}
		return "NVARCHAR(4000)" // ANY
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
* isLobType: Reports whether a column of tp cannot be an index key (MAX types) or is too wide for one.
* @param tp et.TypeData
* @return bool
**/
func isLobType(tp et.TypeData) bool {
	switch msType(tp) {
	case "VARBINARY(MAX)", "NVARCHAR(MAX)", "NVARCHAR(4000)":
		return true
	}
	return false
}

/**
* msDefault: Returns the DEFAULT expression for a type and value.
* @param tp et.TypeData, val any
* @return string
**/
func msDefault(tp et.TypeData, val any) string {
	if val == nil || val == "" {
		return "NULL"
	}
	switch tp {
	case et.INT, et.FLOAT:
		return fmt.Sprintf("%v", val)
	case et.BOOL:
		return msQuoteBool(val)
	case et.DATETIME:
		return "SYSDATETIME()"
	case et.BYTE:
		return "NULL"
	default:
		if isJsonType(tp) {
			return msQuoteJson(val)
		}
		return msQuoteString(val)
	}
}
