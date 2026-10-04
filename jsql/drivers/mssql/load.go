package mssql

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
	return msTableRef(model.Schema, model.Name)
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
* isPrimaryKey: Reports whether name is a primary key of the model.
* @param model *jsql.Model, name string
* @return bool
**/
func isPrimaryKey(model *jsql.Model, name string) bool {
	return slices.ContainsFunc(model.PrimaryKeys, func(pk *jsql.Index) bool { return pk.Name == name })
}

/**
* ddlColumns: Builds the column definitions (only COLUMN types) and the primary key; JSON columns get an
* ISJSON check.
* @param model *jsql.Model
* @return []string
**/
func ddlColumns(model *jsql.Model) []string {
	var cols []string
	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN {
			continue
		}
		name := msIdent(col.Name)
		if col.Name == model.SourceField {
			cols = append(cols, fmt.Sprintf("  %s NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON(%s) = 1)", name, name))
			continue
		}
		line := fmt.Sprintf("  %s %s", name, msType(col.TypeData))
		if isPrimaryKey(model, col.Name) {
			line += " NOT NULL"
		} else {
			line += " DEFAULT " + msDefault(col.TypeData, col.Default)
		}
		if isJsonType(col.TypeData) {
			line += fmt.Sprintf(" CHECK (ISJSON(%s) = 1)", name)
		}
		cols = append(cols, line)
	}
	if len(model.PrimaryKeys) > 0 {
		keys := make([]string, len(model.PrimaryKeys))
		for i, k := range model.PrimaryKeys {
			keys[i] = msIdent(k.Name)
		}
		cols = append(cols, fmt.Sprintf("  CONSTRAINT %s PRIMARY KEY (%s)", msObjectName(model.Schema, model.Name, "pkey"), strings.Join(keys, ", ")))
	}
	return cols
}

/**
* ddlIndexes: Builds UNIQUE INDEX and INDEX statements, skipping columns already indexed and MAX/wide
* columns, which cannot be index keys.
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
		if slices.Contains(indexed, u.Name) || isLobType(columnType(model, u.Name)) {
			continue
		}
		stmts = append(stmts, fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s)", msObjectName(model.Name, u.Name, "key"), table, msIdent(u.Name)))
		indexed = append(indexed, u.Name)
	}
	for _, idx := range model.Indexes {
		if slices.Contains(indexed, idx.Name) || isLobType(columnType(model, idx.Name)) {
			continue
		}
		stmts = append(stmts, fmt.Sprintf("CREATE INDEX %s ON %s (%s)", msObjectName(model.Name, idx.Name, "idx"), table, msIdent(idx.Name)))
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
			locals[i] = msIdent(local)
			foreigns[i] = msIdent(fk.Keys[local])
		}
		stmt := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
			table, msObjectName("fk", model.Schema, model.Name, fk.To.Name), strings.Join(locals, ", "),
			msTableRef(fk.To.Schema, fk.To.Name), strings.Join(foreigns, ", "))
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
* ExistModel: Returns true when the model's table exists.
* @param db *sql.DB, model *jsql.Model
* @return bool, error
**/
func (s *Mssql) ExistModel(db *sql.DB, model *jsql.Model, timeout ...time.Duration) (bool, error) {
	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()

	table := ddlTable(model)
	query := fmt.Sprintf("SELECT CASE WHEN OBJECT_ID(%s, N'U') IS NULL THEN 'false' ELSE 'true' END AS [exists]", msQuoteText(table))
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
func (s *Mssql) Drop(model *jsql.Model, timeout ...time.Duration) error {
	if model.Db() == nil || model.SqlDB() == nil {
		return errors.New(jsql.MSG_DB_IS_NIL)
	}

	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()

	db := model.SqlDB()
	table := ddlTable(model)
	query := fmt.Sprintf("IF OBJECT_ID(%s, N'U') IS NOT NULL DROP TABLE %s", msQuoteText(table), table)
	_, err := db.ExecContext(ctx, query)
	return err
}

/**
* Load: Generates the DDL of the model as one batch: the schema (through EXEC, since CREATE SCHEMA must be
* alone in its batch), CREATE TABLE with its primary key, unique indexes, indexes and foreign keys.
* @param model *jsql.Model
* @return string, error
**/
func (s *Mssql) Load(model *jsql.Model, timeout ...time.Duration) (string, error) {
	table := ddlTable(model)
	cols := ddlColumns(model)
	if len(cols) == 0 {
		return "", fmt.Errorf("model %s has no columns", model.Table)
	}

	stmts := []string{}
	if model.Schema != "" {
		schema := sanitizeIdent(model.Schema)
		stmts = append(stmts, fmt.Sprintf("IF SCHEMA_ID(%s) IS NULL EXEC(%s)", msQuoteText(schema), msQuoteText("CREATE SCHEMA "+msIdent(schema))))
	}
	stmts = append(stmts, fmt.Sprintf("IF OBJECT_ID(%s, N'U') IS NULL CREATE TABLE %s (\n%s\n)", msQuoteText(table), table, strings.Join(cols, ",\n")))
	stmts = append(stmts, ddlIndexes(model, table)...)
	stmts = append(stmts, ddlForeignKeys(model, table)...)
	return strings.Join(stmts, ";\n") + ";", nil
}
