package lib

import (
	"strings"

	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/strs"
)

// DDL Data Definition Language
// This package contains the functions to definition data elements in the database

// Sqlite default values
func ddlDefault(col *linq.Column) string {
	var result string
	switch col.TypeData {
	case linq.TpKey:
		result = `'-1'`
	case linq.TpText:
		result = `''`
	case linq.TpMemo:
		result = `''`
	case linq.TpNumber:
		result = `0`
	case linq.TpDate:
		result = `NOW()`
	case linq.TpCheckbox:
		result = `FALSE`
	case linq.TpRollup:
		result = `''`
	case linq.TpCreatedTime:
		result = `NOW()`
	case linq.TpCreatedBy:
		result = `'{ "_id": "", "name": "" }'`
	case linq.TpLastEditedTime:
		result = `NOW()`
	case linq.TpLastEditedBy:
		result = `'{ "_id": "", "name": "" }'`
	case linq.TpStatus:
		result = `'{ "_id": "0", "main": "State", "name": "Activo" }'`
	case linq.TpPerson:
		result = `'{ "_id": "", "name": "" }'`
	case linq.TpFile:
		result = `''`
	case linq.TpURL:
		result = `''`
	case linq.TpEmail:
		result = `''`
	case linq.TpPhone:
		result = `''`
	case linq.TpFormula:
		result = `''`
	case linq.TpSelect:
		result = `''`
	case linq.TpMultiSelect:
		result = `''`
	case linq.TpSource:
		result = `'{}'`
	case linq.TpJson:
		result = `'{}'`
	case linq.TpArray:
		result = `'[]'`
	case linq.TpSerie:
		result = `0`
	default:
		val := col.Default
		result = strs.Format(`%v`, js.Quote(val))
	}

	return strs.Append("DEFAULT", result, " ")
}

// Sqlite type ddl
func ddlType(col *linq.Column) string {
	switch col.TypeData {
	case linq.TpNumber:
		return "REAL"
	case linq.TpDate:
		return "TIMESTAMP"
	case linq.TpCheckbox:
		return "INTEGER"
	case linq.TpCreatedTime:
		return "TIMESTAMP"
	case linq.TpLastEditedTime:
		return "TIMESTAMP"
	case linq.TpSerie:
		return "INTEGER"
	default:
		return "TEXT"
	}
}

// Sqlite column ddl
func ddlColumn(col *linq.Column) string {
	var result string
	var def string

	result = ddlPrimaryKey(col)
	def = ddlDefault(col)
	result = strs.Append(def, result, " ")
	def = ddlType(col)
	result = strs.Append(def, result, " ")
	result = strs.Append(col.Up(), result, " ")

	return result
}

// Sqlite index ddl
func ddlIndex(col *linq.Column) string {
	name := strs.Format(`%v_%v_IDX`, strs.Uppcase(col.Table()), col.Up())
	name = strs.Replace(name, "-", "_")
	name = strs.Replace(name, ".", "_")
	return strs.Format(`CREATE INDEX IF NOT EXISTS %v ON %v(%v);`, name, strs.Uppcase(col.Table()), col.Up())
}

// Sqlite unique index ddl
func ddlUnique(col *linq.Column) string {
	name := strs.Format(`%v_%v_IDX`, strs.Uppcase(col.Table()), col.Up())
	name = strs.Replace(name, "-", "_")
	name = strs.Replace(name, ".", "_")
	return strs.Format(`CREATE UNIQUE INDEX IF NOT EXISTS %v ON %v(%v);`, name, strs.Uppcase(col.Table()), col.Up())
}

// Sqlite primary key ddl
func ddlPrimaryKey(col *linq.Column) string {
	if col.PrimaryKey {
		return "PRIMARY KEY"
	}

	return ""
}

// Sqlite ForeignKey ddl
func ddlForeignKeys(model *linq.Model) string {
	var result string
	for _, ref := range model.ForeignKey {
		fkey := strings.Join(ref.ForeignKey, ", ")
		pKey := strings.Join(ref.ParentKey, ", ")
		def := strs.Format(`FOREIGN KEY(%s) REFERENCES %s(%s)`, fkey, ref.Parent.Table, pKey)
		result = strs.Append(result, def, ",\n")
	}

	return result
}

// Sqlite table ddl
func ddlTable(model *linq.Model) string {
	var result string
	var columns string
	var indexs string
	for _, col := range model.Columns {
		if col.TypeColumn == linq.TpColumn {
			def := ddlColumn(col)
			columns = strs.Append(def, columns, ",\n")
			if col.PrimaryKey {
				continue
			} else if col.Unique {
				def = ddlUnique(col)
				indexs = strs.Append(def, indexs, "\n")
			} else if col.Indexed {
				def = ddlIndex(col)
				indexs = strs.Append(def, indexs, "\n")
			}
		}
	}
	foreign := ddlForeignKeys(model)
	columns = strs.Append(columns, foreign, ",\n")
	table := strs.Format("CREATE TABLE IF NOT EXISTS %s (\n%s);", model.Table, columns)
	result = strs.Append(result, table, "\n")
	result = strs.Append(result, indexs, "\n")

	return result
}
