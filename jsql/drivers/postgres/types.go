package postgres

import (
	"fmt"

	"github.com/cgalvisleon/et/jsql"
)

/**
* pgType: Maps a jsql TypeData to the corresponding PostgreSQL column type.
* @param tp jsql.TypeData
* @return string
**/
func pgType(tp jsql.TypeData) string {
	switch tp {
	case jsql.INT:
		return "BIGINT"
	case jsql.FLOAT:
		return "DOUBLE PRECISION"
	case jsql.KEY:
		return "VARCHAR(80)"
	case jsql.TEXT:
		return "VARCHAR(255)"
	case jsql.MEMO:
		return "TEXT"
	case jsql.JSON:
		return "JSONB"
	case jsql.DATETIME:
		return "TIMESTAMP"
	case jsql.BOOLEAN:
		return "BOOLEAN"
	case jsql.BYTES:
		return "BYTEA"
	case jsql.GEOMETRY:
		return "JSONB"
	case jsql.EMBEDDING:
		return "VECTOR"
	default: // ANY
		return "TEXT"
	}
}

/**
* pgDefault: Returns the SQL DEFAULT expression for a given TypeData and value.
* @param tp jsql.TypeData
* @param val any
* @return string
**/
func pgDefault(tp jsql.TypeData, val any) string {
	if val == nil || val == "" {
		return "NULL"
	}
	switch tp {
	case jsql.INT, jsql.FLOAT:
		return fmt.Sprintf("%v", val)
	case jsql.BOOLEAN:
		return fmt.Sprintf("%v", val)
	case jsql.JSON:
		return fmt.Sprintf("'%v'::jsonb", val)
	case jsql.DATETIME:
		return "NOW()"
	default:
		return fmt.Sprintf("'%v'", val)
	}
}
