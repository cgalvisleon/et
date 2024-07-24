package lib

import (
	"strings"

	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/strs"
)

/**
* DDL functions to support a models
**/

/**
* Return ddl schema
* @param schema *linq.Schema
* @return string
**/
func ddlSchema(name string) string {
	name = strs.Lowcase(name)
	return strs.Format(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"; CREATE EXTENSION IF NOT EXISTS pgcrypto; CREATE SCHEMA IF NOT EXISTS "%s";`, name)
}

/**
* Return ddl column
* @param col *linq.Column
* @return string
**/
func ddlColumn(col *linq.Column) string {
	var result string
	var def string

	result = ddlDefault(col)
	def = ddlType(col)
	result = strs.Append(def, result, " ")
	result = strs.Append(col.Up(), result, " ")

	return result
}

/**
* Return ddl index
* @param col *linq.Column
* @return string
**/
func ddlIndex(col *linq.Column) string {
	name := strs.Format(`%v_%v_IDX`, strs.Uppcase(col.Table()), col.Up())
	name = strs.Replace(name, "-", "_")
	name = strs.Replace(name, ".", "_")
	return strs.Format(`CREATE INDEX IF NOT EXISTS %v ON %v(%v);`, name, col.Table(), col.Up())
}

/**
* Return ddl unique
* @param col *linq.Column
* @return string
**/
func ddlUnique(col *linq.Column) string {
	name := strs.Format(`%v_%v_IDX`, strs.Uppcase(col.Table()), col.Up())
	name = strs.Replace(name, "-", "_")
	name = strs.Replace(name, ".", "_")
	return strs.Format(`CREATE UNIQUE INDEX IF NOT EXISTS %v ON %v(%v);`, name, col.Table(), col.Up())
}

/**
* ddlPrimaryKey return PrimaryKey ddl
* @param col *linq.Column
* @return string
**/
func ddlPrimaryKey(model *linq.Model) string {
	primaryKeys := func() []string {
		var result []string
		for _, v := range model.PrimaryKeys {
			result = append(result, v.Name)
		}

		return result
	}

	return strs.Format(`PRIMARY KEY (%s)`, strings.Join(primaryKeys(), ", "))
}

/**
* ddlForeignKeys return ForeignKeys ddl
* @param model *linq.Model
* @return string
**/
func ddlForeignKeys(model *linq.Model) string {
	var result string
	for _, ref := range model.ForeignKey {
		key := strs.Replace(model.Table, ".", "_") + "_" + strings.Join(ref.ForeignKey, "_")
		key = strs.Replace(key, "-", "_") + "_fkey"
		key = strs.Lowcase(key)
		return strs.Format(`ALTER TABLE IF EXISTS %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s);`, model.Table, strs.Uppcase(key), strings.Join(ref.ForeignKey, ", "), ref.Parent.Table, strings.Join(ref.ParentKey, ", "))
	}

	return result
}

/**
* ddlSetSync return Sync ddl
* @param model *linq.Model
* @return string
**/
func ddlSetSync(model *linq.Model) string {
	result := linq.SQLDDL(`
	DROP TRIGGER IF EXISTS SYNC_INSERT ON $1 CASCADE;
	CREATE TRIGGER SYNC_INSERT
	BEFORE INSERT ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE core.SYNC_INSERT();

	DROP TRIGGER IF EXISTS SYNC_UPDATE ON $1 CASCADE;
	CREATE TRIGGER SYNC_UPDATE
	BEFORE UPDATE ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE core.SYNC_UPDATE();

	DROP TRIGGER IF EXISTS SYNC_DELETE ON $1 CASCADE;
	CREATE TRIGGER SYNC_DELETE
	BEFORE DELETE ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE core.SYNC_DELETE();`, model.Table)

	result = strs.Replace(result, "\t", "")

	return result
}

/**
* ddlSetRecycling return Recycling ddl
* @param model *linq.Model
* @return string
**/
func ddlSetRecycling(model *linq.Model) string {
	result := linq.SQLDDL(`	
	DROP TRIGGER IF EXISTS RECYCLING ON $1 CASCADE;
	CREATE TRIGGER RECYCLING
	AFTER UPDATE ON $1
	FOR EACH ROW WHEN (OLD._STATE!=NEW._STATE)
	EXECUTE PROCEDURE core.RECYCLING_UPDATE();`, model.Table)

	result = strs.Replace(result, "\t", "")

	return result
}

/**
* ddlTable return Table ddl
* @param model *linq.Model
* @return string
**/
func ddlTable(model *linq.Model) string {
	var result string
	var columns string
	var indexs string
	var primaryKeys string
	var uniqueKeys string

	appedColumns := func(def string) {
		columns = strs.Append(columns, def, ",\n")
	}

	appendIndex := func(def string) {
		indexs = strs.Append(indexs, def, "\n")
	}

	appendUniqueKey := func(def string) {
		uniqueKeys = strs.Append(uniqueKeys, def, ", ")
	}

	for _, col := range model.Columns {
		if col.TypeColumn == linq.TpColumn {
			def := ddlColumn(col)
			appedColumns(def)
			if col.Unique && !col.PrimaryKey {
				def = ddlUnique(col)
				appendUniqueKey(def)
			} else if col.Indexed {
				def = ddlIndex(col)
				appendIndex(def)
			}
		}
	}
	columns = strs.Append(columns, ",", "")
	columns = strs.Append(columns, ddlPrimaryKey(model), "\n")
	result = ddlSchema(model.Schema.Name)
	table := strs.Format("\nCREATE TABLE IF NOT EXISTS %s (\n%s);", model.Table, columns)
	result = strs.Append(result, table, "\n")
	result = strs.Append(result, primaryKeys, "\n")
	result = strs.Append(result, uniqueKeys, "\n")
	result = strs.Append(result, indexs, "\n\n")
	foreign := ddlForeignKeys(model)
	result = strs.Append(result, foreign, "\n\n")
	sync := ddlSetSync(model)
	result = strs.Append(result, sync, "\n\n")
	if model.ColumnStatus != nil {
		recycle := ddlSetRecycling(model)
		result = strs.Append(result, recycle, "\n\n")
	}
	model.DDL = result

	return result
}

/**
* ddlTableRename return TableRename ddl
* @param model *linq.Model
* @return string
**/
func ddlTableRename(model *linq.Model, name string) string {
	newName := model.Schema.Name + "." + name
	return strs.Format(`ALTER TABLE IF EXISTS %s RENAME TO %s;`, model.Table, newName)
}

/**
* ddlTableDrop return TableDrop ddl
* @param model *linq.Model
* @return string
**/
func ddlTableDrop(schema, name string) string {
	dropName := schema + "." + name
	return strs.Format(`DROP TABLE IF EXISTS %s;`, dropName)
}
