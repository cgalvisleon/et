package jsql

import (
	"testing"

	"github.com/cgalvisleon/et/et"
)

/**
* TestTypeDataOf: The SQL types each driver's catalog gives map to the et.TypeData closest to them.
* @param t *testing.T
**/
func TestTypeDataOf(t *testing.T) {
	cases := map[string]et.TypeData{
		"character varying": et.TEXT, "varchar(80)": et.TEXT, "nvarchar": et.TEXT, "VARCHAR2": et.TEXT,
		"text": et.MEMO, "longtext": et.MEMO, "nvarchar(max)": et.MEMO, "CLOB": et.MEMO,
		"bigint": et.INT, "integer": et.INT, "INTEGER": et.INT, "int(11)": et.INT, "NUMBER(19,0)": et.INT, "number(10)": et.INT,
		"double precision": et.FLOAT, "REAL": et.FLOAT, "numeric": et.FLOAT, "NUMBER(10,2)": et.FLOAT, "NUMBER": et.FLOAT, "float": et.FLOAT,
		"boolean": et.BOOL, "tinyint(1)": et.BOOL, "bit": et.BOOL,
		"timestamp without time zone": et.DATETIME, "datetime(6)": et.DATETIME, "datetime2": et.DATETIME, "TIMESTAMP(6)": et.DATETIME, "date": et.DATETIME,
		"jsonb": et.JSON, "json": et.JSON,
		"bytea": et.BYTE, "BLOB": et.BYTE, "longblob": et.BYTE, "varbinary(max)": et.BYTE, "RAW": et.BYTE,
		"array": et.ARRAY, "text[]": et.ARRAY,
		"interval": et.TEXT, "": et.ANY,
	}
	for sqlType, want := range cases {
		if got := typeDataOf(sqlType); got != want {
			t.Errorf("typeDataOf(%q) = %v, want %v", sqlType, got, want)
		}
	}
}
