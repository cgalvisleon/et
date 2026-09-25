package sqlite

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
)

/**
* sqliteType: Maps an et.TypeData to the corresponding SQLite column type.
* @param tp et.TypeData
* @return string
**/
func sqliteType(tp et.TypeData) string {
	switch tp {
	case et.BYTE:
		return "BLOB"
	case et.INT, et.BOOL:
		return "INTEGER"
	case et.FLOAT:
		return "REAL"
	case et.KEY, et.TEXT, et.MEMO, et.DATETIME:
		return "TEXT"
	case et.JSON, et.ARRAY, et.ARRAY_JSON, et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME, et.VAL_BETWEEN:
		// SQLite no tiene arreglos: se guardan como JSON en texto
		return "TEXT"
	default: // ANY
		return "TEXT"
	}
}

/**
* sqliteDefault: Returns the SQL DEFAULT expression for a given TypeData and value.
* @param tp et.TypeData, val any
* @return string
**/
func sqliteDefault(tp et.TypeData, val any) string {
	if val == nil || val == "" {
		return "NULL"
	}
	switch tp {
	case et.INT, et.FLOAT:
		return fmt.Sprintf("%v", val)
	case et.BOOL:
		return fmt.Sprintf("%v", val)
	case et.JSON:
		return "'{}'"
	case et.ARRAY, et.ARRAY_JSON, et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME:
		return "'[]'"
	case et.DATETIME:
		return "CURRENT_TIMESTAMP"
	default:
		return fmt.Sprintf("'%v'", val)
	}
}
