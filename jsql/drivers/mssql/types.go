package mssql

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/jsql"
)

/**
* msType: Maps a jsql TypeData to the corresponding SQL Server column type.
* JSON is stored as NVARCHAR(MAX) since SQL Server has no native JSON column type.
* @param tp jsql.TypeData
* @return string
**/
func msType(tp jsql.TypeData) string {
	switch tp {
	case jsql.INT:
		return "BIGINT"
	case jsql.FLOAT:
		return "FLOAT"
	case jsql.KEY:
		return "NVARCHAR(80)"
	case jsql.TEXT:
		return "NVARCHAR(255)"
	case jsql.MEMO:
		return "NVARCHAR(MAX)"
	case jsql.JSON:
		return "NVARCHAR(MAX)"
	case jsql.DATETIME:
		return "DATETIME2(7)"
	case jsql.BOOLEAN:
		return "BIT"
	case jsql.BYTES:
		return "VARBINARY(MAX)"
	case jsql.GEOMETRY:
		return "NVARCHAR(MAX)"
	case jsql.EMBEDDING:
		return "NVARCHAR(MAX)"
	default:
		return "NVARCHAR(255)"
	}
}

/**
* msDefault: Returns the SQL DEFAULT expression for a given TypeData and value.
* String literals use the N'' prefix for Unicode. JSON defaults to N'{}' .
* @param tp jsql.TypeData
* @param val any
* @return string
**/
func msDefault(tp jsql.TypeData, val any) string {
	if val == nil || val == "" {
		return "NULL"
	}
	switch tp {
	case jsql.INT, jsql.FLOAT:
		return fmt.Sprintf("%v", val)
	case jsql.BOOLEAN:
		return fmt.Sprintf("%v", val)
	case jsql.JSON:
		return "N'{}'"
	case jsql.DATETIME:
		return "GETUTCDATE()"
	default:
		return fmt.Sprintf("N'%v'", val)
	}
}

/**
* msJsonPath: Converts a '->' separated field path into a SQL Server JSON path literal ('$.a.b.c').
* @param field string
* @return string
**/
func msJsonPath(field string) string {
	parts := strings.Split(field, "->")
	return "'$." + strings.Join(parts, ".") + "'"
}

/**
* msJsonValue: Builds a JSON_VALUE (or CAST thereof) expression for an ATTRIB scalar field.
* JSON_VALUE returns NVARCHAR; numeric/bool types are wrapped in CAST.
* @param col string, field string, tp jsql.TypeData
* @return string
**/
func msJsonValue(col, field string, tp jsql.TypeData) string {
	path := msJsonPath(field)
	base := fmt.Sprintf("JSON_VALUE(%s, %s)", col, path)
	switch tp {
	case jsql.INT:
		return fmt.Sprintf("CAST(%s AS BIGINT)", base)
	case jsql.FLOAT:
		return fmt.Sprintf("CAST(%s AS FLOAT)", base)
	case jsql.BOOLEAN:
		return fmt.Sprintf("CAST(%s AS BIT)", base)
	case jsql.DATETIME:
		return fmt.Sprintf("CAST(%s AS DATETIME2(7))", base)
	default:
		return base
	}
}
