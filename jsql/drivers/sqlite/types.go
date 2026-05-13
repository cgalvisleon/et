package sqlite

import (
	"fmt"

	"github.com/cgalvisleon/et/jsql"
)

/**
* sqType: Maps a jsql TypeData to the corresponding SQLite column type.
* SQLite uses TEXT for JSON/datetime, INTEGER for booleans, BLOB for bytes.
* @param tp jsql.TypeData
* @return string
**/
func sqType(tp jsql.TypeData) string {
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
	default:
		return "TEXT"
	}
}

/**
* sqDefault: Returns the SQL DEFAULT expression for a given TypeData and value.
* @param tp jsql.TypeData
* @param val any
* @return string
**/
func sqDefault(tp jsql.TypeData, val any) string {
	if val == nil || val == "" {
		return "NULL"
	}
	switch tp {
	case jsql.INT, jsql.FLOAT:
		return fmt.Sprintf("%v", val)
	case jsql.BOOLEAN:
		return fmt.Sprintf("%v", val)
	case jsql.JSON:
		return fmt.Sprintf("'%v'", val)
	case jsql.DATETIME:
		return "CURRENT_TIMESTAMP"
	default:
		return fmt.Sprintf("'%v'", val)
	}
}

/**
* sqAttribCast: Returns the SQLite CAST target type when the ATTRIB TypeData
* requires a non-text result from json_extract.
* Returns empty string for TEXT types (no cast needed).
* @param tp jsql.TypeData
* @return string
**/
func sqAttribCast(tp jsql.TypeData) string {
	switch tp {
	case jsql.INT:
		return "INTEGER"
	case jsql.FLOAT:
		return "REAL"
	case jsql.BOOLEAN:
		return "INTEGER"
	default:
		return ""
	}
}
