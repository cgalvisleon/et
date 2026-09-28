package mysql

import (
	"database/sql"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* ddlTable: Sets model.Table (schema.name) and returns the quoted table reference.
* @param model *jsql.Model
* @return string
**/
func ddlTable(model *jsql.Model) string {
	if model.Schema != "" {
		model.Table = fmt.Sprintf("%s.%s", model.Schema, model.Name)
	} else {
		model.Table = model.Name
	}
	return myTableRef(model.Schema, model.Name)
}

/**
* columnType: Returns the TypeData of a real column of the model, or et.ANY.
* @param model *jsql.Model, name string
* @return et.TypeData
**/
func columnType(model *jsql.Model, name string) et.TypeData {
	for _, col := range model.Columns {
		if col.Name == name && col.TypeColumn == jsql.COLUMN {
			return col.TypeData
		}
	}
	return et.ANY
}

/**
* ddlColumns: Builds the column definitions (only COLUMN types); the SourceField is a JSON column.
* @param model *jsql.Model
* @return []string
**/
func ddlColumns(model *jsql.Model) []string {
	var cols []string
	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN {
			continue
		}
		name := myIdent(col.Name)
		if col.Name == model.SourceField {
			cols = append(cols, fmt.Sprintf("  %s JSON DEFAULT (JSON_OBJECT())", name))
			continue
		}
		notNull := ""
		if slices.ContainsFunc(model.PrimaryKeys, func(pk *jsql.Index) bool { return pk.Name == col.Name }) {
			notNull = " NOT NULL"
		}
		def := myDefault(col.TypeData, col.Default)
		if notNull != "" && def == "NULL" {
			// A primary key cannot default to NULL.
			cols = append(cols, fmt.Sprintf("  %s %s%s", name, myType(col.TypeData), notNull))
			continue
		}
		cols = append(cols, fmt.Sprintf("  %s %s%s DEFAULT %s", name, myType(col.TypeData), notNull, def))
	}
	if len(model.PrimaryKeys) > 0 {
		keys := make([]string, len(model.PrimaryKeys))
		for i, k := range model.PrimaryKeys {
			keys[i] = myIdent(k.Name)
		}
		cols = append(cols, fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(keys, ", ")))
	}
	return cols
}

/**
* ddlIndexes: Builds UNIQUE INDEX and INDEX statements, skipping columns already indexed and TEXT/BLOB/JSON
* columns (MySQL needs a prefix length to index them).
* @param model *jsql.Model, table string
* @return []string
**/
func ddlIndexes(model *jsql.Model, table string) []string {
	stmts := make([]string, 0)
	indexed := make([]string, 0)
	if len(model.PrimaryKeys) == 1 {
		indexed = append(indexed, model.PrimaryKeys[0].Name)
	}
	for _, u := range model.Unique {
		if slices.Contains(indexed, u.Name) || isBlobType(columnType(model, u.Name)) {
			continue
		}
		stmts = append(stmts, fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s)", myObjectName(model.Name, u.Name, "key"), table, myIdent(u.Name)))
		indexed = append(indexed, u.Name)
	}
	for _, idx := range model.Indexes {
		if slices.Contains(indexed, idx.Name) || isBlobType(columnType(model, idx.Name)) {
			continue
		}
		stmts = append(stmts, fmt.Sprintf("CREATE INDEX %s ON %s (%s)", myObjectName(model.Name, idx.Name, "idx"), table, myIdent(idx.Name)))
		indexed = append(indexed, idx.Name)
	}
	return stmts
}

/**
* ddlForeignKeys: Builds ALTER TABLE … ADD CONSTRAINT … FOREIGN KEY statements.
* @param model *jsql.Model, table string
* @return []string
**/
func ddlForeignKeys(model *jsql.Model, table string) []string {
	stmts := make([]string, 0, len(model.ForeignKeys))
	for _, fk := range model.ForeignKeys {
		if fk.To == nil || len(fk.Keys) == 0 {
			continue
		}
		localCols := make([]string, 0, len(fk.Keys))
		for local := range fk.Keys {
			localCols = append(localCols, local)
		}
		sort.Strings(localCols)
		locals := make([]string, len(localCols))
		foreigns := make([]string, len(localCols))
		for i, local := range localCols {
			locals[i] = myIdent(local)
			foreigns[i] = myIdent(fk.Keys[local])
		}
		stmt := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
			table, myObjectName("fk", model.Name, fk.To.Name), strings.Join(locals, ", "),
			myTableRef(fk.To.Schema, fk.To.Name), strings.Join(foreigns, ", "))
		if fk.OnDeleteCascade {
			stmt += " ON DELETE CASCADE"
		}
		if fk.OnUpdateCascade {
			stmt += " ON UPDATE CASCADE"
		}
		stmts = append(stmts, stmt)
	}
	return stmts
}

/**
* ExistModel: Returns true when the model's table exists in its schema (database) or in the current one.
* @param db *sql.DB, model *jsql.Model
* @return bool, error
**/
func (s *Mysql) ExistModel(db *sql.DB, model *jsql.Model) (bool, error) {
	ddlTable(model)
	schema := "DATABASE()"
	if model.Schema != "" {
		schema = myQuoteText(sanitizeIdent(model.Schema))
	}
	query := fmt.Sprintf("SELECT CASE WHEN EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = %s AND table_name = %s) THEN 'true' ELSE 'false' END AS `exists`",
		schema, myQuoteText(sanitizeIdent(model.Name)))
	rows, err := db.Query(query)
	if err != nil {
		return false, err
	}
	items := jsql.RowsToItems(rows)
	if items.Count == 0 {
		return false, nil
	}
	return items.Bool(0, "exists"), nil
}

/**
* Load: Generates the DDL of the model as one batch: CREATE SCHEMA (a MySQL database), CREATE TABLE with
* its primary key, unique indexes, indexes and foreign keys.
* @param model *jsql.Model
* @return string, error
**/
func (s *Mysql) Load(model *jsql.Model) (string, error) {
	table := ddlTable(model)
	cols := ddlColumns(model)
	if len(cols) == 0 {
		return "", fmt.Errorf("model %s has no columns", model.Table)
	}

	stmts := []string{}
	if model.Schema != "" {
		stmts = append(stmts, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", myIdent(model.Schema)))
	}
	stmts = append(stmts, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n)", table, strings.Join(cols, ",\n")))
	stmts = append(stmts, ddlIndexes(model, table)...)
	stmts = append(stmts, ddlForeignKeys(model, table)...)
	return strings.Join(stmts, ";\n") + ";", nil
}
