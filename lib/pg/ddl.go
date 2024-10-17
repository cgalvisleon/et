package lib

import (
	"strings"

	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/strs"
)

/**
* DefineSql return DDL sql to define a model
* @param m *linq.Model
* @return string
**/
func (d *Postgres) DefineSql(m *linq.Model) string {
	m = ddlTable(m)
	m = ddlIndexes(m)
	m = ddlForeignKeys(m)
	m = ddlSetObject(m)
	m = ddlSetRecycling(m)

	result := m.DDL.Table
	result = strs.Append(result, m.DDL.Indexes, "\n")
	result = strs.Append(result, m.DDL.ForeignKeys, "\n")
	result = strs.Append(result, m.DDL.Objects, "\n")
	result = strs.Append(result, m.DDL.Recycling, "\n")

	return result
}

/**
* MutationSql return DDL mutation the sql to mutate
* @param l *linq.Linq
* @return string
**/
func (d *Postgres) MutationSql(m *linq.Model) string {

	return ""
}

/**
* ddlTable return Table ddl
* @param model *linq.Model
* @return *linq.Model
**/
func ddlTable(model *linq.Model) *linq.Model {
	var result string
	var columns string

	model.DefineColumn(linq.IdTField.Low(), "UUId", linq.TpKey, "-1")

	appedColumns := func(def string) {
		columns = strs.Append(columns, def, ",\n")
	}

	for _, col := range model.Columns {
		if col.TypeColumn == linq.TpColumn {
			def := ddlColumn(col)
			appedColumns(def)
		}
	}
	columns = strs.Append(columns, ",", "")
	columns = strs.Append(columns, ddlPrimaryKey(model), "\n")
	result = ddlSchema(model.Schema.Name)
	table := strs.Format("\nCREATE TABLE IF NOT EXISTS %s (\n%s);", model.Table, columns)
	result = strs.Append(result, table, "\n")
	model.DDL.Table = result

	return model
}

/**
* ddlIndexes return Index ddl
* @param model *linq.Model
* @return *linq.Model
**/
func ddlIndexes(model *linq.Model) *linq.Model {
	var result string
	var indexs string
	var uniqueKeys string

	appendIndex := func(def string) {
		indexs = strs.Append(indexs, def, "\n")
	}

	appendUniqueKey := func(def string) {
		uniqueKeys = strs.Append(uniqueKeys, def, ", ")
	}

	for _, col := range model.Columns {
		if col.TypeColumn == linq.TpColumn {
			if col.Unique && !col.PrimaryKey {
				def := ddlUnique(col)
				appendUniqueKey(def)
			} else if col.Indexed {
				def := ddlIndex(col)
				appendIndex(def)
			}
		}
	}

	result = strs.Append(result, uniqueKeys, "\n")
	result = strs.Append(result, indexs, "\n")
	model.DDL.Indexes = result

	return model
}

/**
* ddlForeignKeys return ForeignKeys ddl
* @param model *linq.Model
* @return *linq.Model
**/
func ddlForeignKeys(model *linq.Model) *linq.Model {
	var result string
	for _, ref := range model.ForeignKey {
		key := strs.Replace(model.Table, ".", "_") + "_" + strings.Join(ref.ForeignKey, "_")
		key = strs.Replace(key, "-", "_") + "_fkey"
		key = strs.Lowcase(key)
		result = strs.Append(result, strs.Format(`ALTER TABLE IF EXISTS %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s);`, model.Table, strs.Uppcase(key), strings.Join(ref.ForeignKey, ", "), ref.Parent.Table, strings.Join(ref.ParentKey, ", ")), "\n")
	}
	model.DDL.ForeignKeys = result

	return model
}

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
* ddlSetObject return Sync ddl
* @param model *linq.Model
* @return *linq.Model
**/
func ddlSetObject(model *linq.Model) *linq.Model {
	if model.ColumnSource == nil {
		return model
	}

	result := linq.SQLDDL(`
	DROP TRIGGER IF EXISTS OBJECTS_INSERT ON $1 CASCADE;
	CREATE TRIGGER OBJECTS_INSERT
	BEFORE INSERT ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE core.OBJECTS_INSERT();

	DROP TRIGGER IF EXISTS OBJECTS_UPDATE ON $1 CASCADE;
	CREATE TRIGGER OBJECTS_UPDATE
	BEFORE UPDATE ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE core.OBJECTS_UPDATE();

	DROP TRIGGER IF EXISTS OBJECTS_DELETE ON $1 CASCADE;
	CREATE TRIGGER OBJECTS_DELETE
	BEFORE DELETE ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE core.OBJECTS_DELETE();`, model.Table)

	model.DDL.Objects = strs.Replace(result, "\t", "")

	return model
}

/**
* ddlSetRecycling return Recycling ddl
* @param model *linq.Model
* @return *linq.Model
**/
func ddlSetRecycling(model *linq.Model) *linq.Model {
	if model.ColumnStatus == nil {
		return model
	}

	result := linq.SQLDDL(`	
	DROP TRIGGER IF EXISTS RECYCLING ON $1 CASCADE;
	CREATE TRIGGER RECYCLING_UPDATE
	AFTER UPDATE ON $1
	FOR EACH ROW WHEN (OLD._STATE!=NEW._STATE)
	EXECUTE PROCEDURE core.RECYCLING_UPDATE();

	DROP TRIGGER IF EXISTS RECYCLING ON $1 CASCADE;
	CREATE TRIGGER RECYCLING_DELETE
	AFTER DELETE ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE core.RECYCLING_DELETE();`, model.Table)

	model.DDL.Recycling = strs.Replace(result, "\t", "")

	return model
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
