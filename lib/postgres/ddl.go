package lib

import (
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/strs"
)

/**
* DDL functions to support a models
**/

/**
* Return default ddl value
* @param col *linq.Column
* @return string
**/
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
	case linq.TpRelation:
		result = `''`
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
	case linq.TpJson:
		result = `'{}'`
	case linq.TpArray:
		result = `'[]'`
	case linq.TpSerie:
		result = `0`
	default:
		val := col.Default
		result = strs.Format(`%v`, et.Quote(val))
	}

	return strs.Append("DEFAULT", result, " ")
}

/**
* Return ddl type
* @param col *linq.Column
* @return string
**/
func ddlType(col *linq.Column) string {
	switch col.TypeData {
	case linq.TpKey, linq.TpRelation, linq.TpRollup, linq.TpStatus, linq.TpPhone, linq.TpSelect, linq.TpMultiSelect:
		return "VARCHAR(80)"
	case linq.TpMemo:
		return "TEXT"
	case linq.TpNumber:
		return "DECIMAL(18, 2)"
	case linq.TpDate:
		return "TIMESTAMP"
	case linq.TpCheckbox:
		return "BOOLEAN"
	case linq.TpCreatedTime:
		return "TIMESTAMP"
	case linq.TpCreatedBy:
		return "JSONB"
	case linq.TpLastEditedTime:
		return "TIMESTAMP"
	case linq.TpLastEditedBy:
		return "JSONB"
	case linq.TpPerson:
		return "JSONB"
	case linq.TpFile:
		return "JSONB"
	case linq.TpURL:
		return "TEXT"
	case linq.TpFormula:
		return "JSONB"
	case linq.TpJson:
		return "JSONB"
	case linq.TpArray:
		return "JSONB"
	case linq.TpSerie:
		return "BIGINT"
	default:
		return "VARCHAR(250)"
	}
}

/**
* Return ddl schema
* @param schema *linq.Schema
* @return string
**/
func ddlSchema(schema *linq.Schema) string {
	return strs.Format(`CREATE SCHEMA IF NOT EXISTS %s;`, schema.Name)
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
	return strs.Format(`CREATE INDEX IF NOT EXISTS %v ON %v(%v);`, name, strs.Uppcase(col.Table()), col.Up())
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
	return strs.Format(`CREATE UNIQUE INDEX IF NOT EXISTS %v ON %v(%v);`, name, strs.Uppcase(col.Table()), col.Up())
}

/**
* ddlPrimaryKey return PrimaryKey ddl
* @param col *linq.Column
* @return string
**/
func ddlPrimaryKey(col *linq.Column) string {
	key := strs.Replace(col.Table(), ".", "_")
	key = strs.Replace(key, "-", "_") + "_pkey"
	key = strs.Lowcase(key)
	return strs.Format(`ALTER TABLE IF EXISTS %s ADD CONSTRAINT %s PRIMARY KEY (%s);`, strs.Uppcase(col.Table()), key, strings.Join(col.PrimaryKeys(), ", "))
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
		return strs.Format(`ALTER TABLE IF EXISTS %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s);`, strs.Uppcase(model.Table), key, strings.Join(ref.ForeignKey, ", "), ref.Parent.Table, strings.Join(ref.ParentKey, ", "))
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
	EXECUTE PROCEDURE linq.SYNC_INSERT();

	DROP TRIGGER IF EXISTS SYNC_UPDATE ON $1 CASCADE;
	CREATE TRIGGER SYNC_UPDATE
	BEFORE UPDATE ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE linq.SYNC_UPDATE();

	DROP TRIGGER IF EXISTS SYNC_DELETE ON $1 CASCADE;
	CREATE TRIGGER SYNC_DELETE
	BEFORE DELETE ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE linq.SYNC_DELETE();`, strs.Uppcase(model.Table))

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
	FOR EACH ROW
	EXECUTE PROCEDURE linq.RECYCLING();

	DROP TRIGGER IF EXISTS ERASE ON $1 CASCADE;
	CREATE TRIGGER ERASE
	AFTER DELETE ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE linq.ERASE();`, strs.Uppcase(model.Table))

	result = strs.Replace(result, "\t", "")

	return result
}

/**
* ddlSetSeries return Series ddl
* @param model *linq.Model
* @return string
**/
func ddlSetSeries(model *linq.Model) string {
	result := linq.SQLDDL(`	
	DROP TRIGGER IF EXISTS SERIES_INSERT ON $1 CASCADE;
	CREATE TRIGGER SERIES_INSERT
	BEFORE INSERT ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE linq.SERIES_INSERT();

	DROP TRIGGER IF EXISTS SERIES_UPDATE ON $1 CASCADE;
	CREATE TRIGGER SERIES_UPDATE
	AFTER UPDATE ON $1
	FOR EACH ROW WHEN (NEW!=OLD)
	EXECUTE PROCEDURE linq.SERIES_UPDATE();`, strs.Uppcase(model.Table))

	result = strs.Replace(result, "\t", "")

	return result
}

/**
* ddlSetModel return SetModel ddl
* @param model *linq.Model
* @return string
**/
func ddlSetModel(model *linq.Model) string {
	schema := model.Schema.Name
	table := model.Name
	definition := model.Definition().ToString()
	result := linq.SQLDDL(`	
	SELECT linq.setmodel('$1', '$2', '$3');`, schema, table, definition)

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
	for _, col := range model.Columns {
		if col.TypeColumn == linq.TpColumn {
			def := ddlColumn(col)
			columns = strs.Append(def, columns, ",\n")
			if col.PrimaryKey {
				def = ddlPrimaryKey(col)
				indexs = strs.Append(def, indexs, "\n")
			} else if col.Unique {
				def = ddlUnique(col)
				indexs = strs.Append(def, indexs, "\n")
			} else if col.Indexed {
				def = ddlIndex(col)
				indexs = strs.Append(def, indexs, "\n")
			}
		}
	}
	schema := ddlSchema(model.Schema)
	result = strs.Append(result, schema, "\n")
	table := strs.Format("CREATE TABLE IF NOT EXISTS %s (\n%s);", model.Table, columns)
	result = strs.Append(result, table, "\n")
	result = strs.Append(result, indexs, "\n\n")
	foreign := ddlForeignKeys(model)
	result = strs.Append(result, foreign, "\n\n")
	sync := ddlSetSync(model)
	result = strs.Append(result, sync, "\n\n")
	recycle := ddlSetRecycling(model)
	result = strs.Append(result, recycle, "\n\n")
	series := ddlSetSeries(model)
	result = strs.Append(result, series, "\n\n")
	model.DDL = result
	define := ddlSetModel(model)
	result = strs.Append(result, define, "\n\n")

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
