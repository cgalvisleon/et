package lib

import (
	"strings"

	"github.com/cgalvisleon/et/js"
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
	str, ok := col.Default.(string)
	if ok {
		switch str {
		case "NOW()":
			return "DEFAULT NOW()"
		case "FALSE":
			return "DEFAULT FALSE"
		case "TRUE":
			return "DEFAULT TRUE"
		case "NULL":
			return "DEFAULT NULL"
		default:
			return strs.Format(`DEFAULT '%v'`, str)
		}
	}
	var result string
	switch col.TypeData {
	case linq.TpKey:
		result = `'-1'`
	case linq.TpText:
		result = `''`
	case linq.TpMemo:
		result = `''`
	case linq.TpInteger:
		result = `0`
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
	case linq.TpData:
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

/**
* Return ddl type
* @param col *linq.Column
* @return string
**/
func ddlType(col *linq.Column) string {
	switch col.TypeData {
	case linq.TpKey, linq.TpRelation, linq.TpRollup, linq.TpStatus, linq.TpPhone, linq.TpSelect, linq.TpMultiSelect, linq.TpCode:
		return "VARCHAR(80)"
	case linq.TpMemo:
		return "TEXT"
	case linq.TpInteger:
		return "BIGINT"
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
	case linq.TpData:
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
