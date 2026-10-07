package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* ddlTable: Returns the quoted table reference of model.Table (schema_name).
* @param model *jsql.Model
* @return string
**/
func ddlTable(model *jsql.Model) string {
	return myIdent(model.Table)
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
			myIdent(fk.To.Table), strings.Join(foreigns, ", "))
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
* ExistModel: Returns true when the model's table (schema_name) exists in the connection database.
* @param db *sql.DB, model *jsql.Model
* @return bool, error
**/
func (s *Mysql) ExistModel(db *sql.DB, model *jsql.Model, timeout ...time.Duration) (bool, error) {
	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()

	query := fmt.Sprintf("SELECT CASE WHEN EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = %s) THEN 'true' ELSE 'false' END AS `exists`",
		myQuoteText(sanitizeIdent(model.Table)))
	rows, err := db.QueryContext(ctx, query)
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
* Drop: Drops the model's table from the database; a missing table is not an error.
* @param model *jsql.Model, timeout ...time.Duration
* @return error
**/
func (s *Mysql) Drop(model *jsql.Model, timeout ...time.Duration) error {
	if model.Db() == nil || model.SqlDB() == nil {
		return errors.New(jsql.MSG_DB_IS_NIL)
	}

	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()

	db := model.SqlDB()
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", ddlTable(model))
	_, err := db.ExecContext(ctx, query)
	return err
}

/**
* Load: Generates the DDL of the model as one batch: CREATE TABLE with its primary key, unique indexes,
* indexes and foreign keys. The schema is not created: the table is schema_name in the connection database.
* @param model *jsql.Model
* @return string, error
**/
func (s *Mysql) Load(model *jsql.Model, timeout ...time.Duration) (string, error) {
	table := ddlTable(model)
	cols := ddlColumns(model)
	if len(cols) == 0 {
		return "", fmt.Errorf("model %s has no columns", model.Table)
	}

	stmts := []string{fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n)", table, strings.Join(cols, ",\n"))}
	stmts = append(stmts, ddlIndexes(model, table)...)
	stmts = append(stmts, ddlForeignKeys(model, table)...)
	return strings.Join(stmts, ";\n") + ";", nil
}
