package sqlite

import (
	"fmt"

	"github.com/cgalvisleon/et/jsql"
)

/**
* sqliteType: Maps a jsql TypeData to the corresponding SQLite column type.
* @param tp jsql.TypeData
* @return string
**/
func sqliteType(tp jsql.TypeData) string {
	switch tp {
	case jsql.INT:
		return "INTEGER"
	case jsql.FLOAT:
		return "REAL"
	case jsql.KEY:
		return "TEXT"
	case jsql.TEXT:
		return "TEXT"
	case jsql.MEMO:
		return "TEXT"
	case jsql.JSON:
		return "TEXT"
	case jsql.DATETIME:
		return "TEXT"
	case jsql.BOOLEAN:
		return "INTEGER"
	case jsql.BYTES:
		return "BLOB"
	case jsql.GEOMETRY:
		return "TEXT"
	case jsql.EMBEDDING:
		return "TEXT"
	default: // ANY
		return "TEXT"
	}
}

/**
* sqliteDefault: Returns the SQL DEFAULT expression for a given TypeData and value.
* @param tp jsql.TypeData
* @param val any
* @return string
**/
func sqliteDefault(tp jsql.TypeData, val any) string {
	if val == nil || val == "" {
		return "NULL"
	}
	switch tp {
	case jsql.INT, jsql.FLOAT:
		return fmt.Sprintf("%v", val)
	case jsql.BOOLEAN:
		return fmt.Sprintf("%v", val)
	case jsql.JSON:
		return "'{}'"
	case jsql.DATETIME:
		return "CURRENT_TIMESTAMP"
	default:
		return fmt.Sprintf("'%v'", val)
	}
}
