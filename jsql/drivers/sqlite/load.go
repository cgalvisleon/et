package sqlite

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/jsql"
)

/**
* ddlTable: Returns the table identifier. SQLite has no schema concept in the
* sense Postgres does, so model.Schema is ignored and the table is unqualified.
* @param model *jsql.Model
* @return string
**/
func ddlTable(model *jsql.Model) string {
	model.Table = model.Name
	return model.Table
}

/**
* ddlColumns: Builds the column definition list for CREATE TABLE.
* @param model *jsql.Model
* @return []string
**/
func ddlColumns(model *jsql.Model) []string {
	var cols []string

	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN {
			continue
		}
		if col.Name == model.SourceField {
			cols = append(cols, fmt.Sprintf("  %s TEXT DEFAULT '{}'", model.SourceField))
			continue
		}
		tp := sqliteType(col.TypeData)
		def := sqliteDefault(col.TypeData, col.Default)
		line := fmt.Sprintf("  %s %s DEFAULT %s", col.Name, tp, def)
		cols = append(cols, line)
	}

	return cols
}

/**
* ddlPrimaryKey: Builds the trailing "PRIMARY KEY (...)" table-constraint clause
* (SQLite cannot add a primary key via ALTER TABLE after creation, so it must be
* embedded inside the CREATE TABLE statement itself), or empty string.
* @param model *jsql.Model
* @return string
**/
func ddlPrimaryKey(model *jsql.Model) string {
	if len(model.PrimaryKeys) == 0 {
		return ""
	}
	keys := make([]string, len(model.PrimaryKeys))
	for i, k := range model.PrimaryKeys {
		keys[i] = k.Name
	}
	return fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(keys, ", "))
}

/**
* ddlForeignKeys: Builds trailing "FOREIGN KEY (...) REFERENCES ..." table-constraint
* clauses (also embedded in CREATE TABLE — SQLite cannot add them afterward).
* Keys map entries are sorted for deterministic output.
* @param model *jsql.Model
* @return []string
**/
func ddlForeignKeys(model *jsql.Model) []string {
	var clauses []string
	for _, fk := range model.ForeignKeys {
		if fk.To == nil || len(fk.Keys) == 0 {
			continue
		}

		foreignTable := fk.To.Name

		localCols := make([]string, 0, len(fk.Keys))
		for local := range fk.Keys {
			localCols = append(localCols, local)
		}
		sort.Strings(localCols)

		foreignCols := make([]string, len(localCols))
		for i, local := range localCols {
			foreignCols[i] = fk.Keys[local]
		}

		clause := fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s (%s)",
			strings.Join(localCols, ", "), foreignTable, strings.Join(foreignCols, ", "))
		if fk.OnDeleteCascade {
			clause += " ON DELETE CASCADE"
		}
		if fk.OnUpdateCascade {
			clause += " ON UPDATE CASCADE"
		}
		clauses = append(clauses, clause)
	}
	return clauses
}

/**
* ddlUnique: Builds UNIQUE INDEX statements.
* @param model *jsql.Model, table string
* @return []string
**/
func ddlUnique(model *jsql.Model, table string) []string {
	stmts := make([]string, 0, len(model.Unique))
	for _, u := range model.Unique {
		idxName := fmt.Sprintf("%s_%s_key", table, u.Name)
		stmts = append(stmts, fmt.Sprintf(
			"CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s);",
			idxName, table, u.Name))
	}
	return stmts
}

/**
* ddlIndexes: Builds CREATE INDEX statements for regular indexes. SQLite indexes
* are always B-tree — there is no USING HASH/BTREE method clause to emit.
* @param model *jsql.Model, table string
* @return []string
**/
func ddlIndexes(model *jsql.Model, table string) []string {
	stmts := make([]string, 0, len(model.Indexes))
	for _, idx := range model.Indexes {
		idxName := fmt.Sprintf("%s_%s_idx", table, idx.Name)
		stmts = append(stmts, fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s ON %s (%s);",
			idxName, table, idx.Name))
	}
	return stmts
}

/**
* ExistModel: Returns true when a table with the given name exists in the database.
* @param db *sql.DB, model *jsql.Model
* @return bool, error
**/
func (s *Sqlite) ExistModel(db *sql.DB, model *jsql.Model) (bool, error) {
	table := ddlTable(model)
	query := `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?);`
	rows, err := db.Query(query, table)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	items := jsql.RowsToItems(rows)
	if items.Count == 0 {
		return false, nil
	}

	return items.Bool(0, "exists"), nil
}

/**
* Load: Generates the DDL SQL to create the table (with PRIMARY KEY and FOREIGN KEY
* constraints embedded inline, since SQLite cannot add them via ALTER TABLE), the
* unique indexes and the regular indexes for the given model. Returns the complete
* DDL as a single string with statements separated by newlines.
* @param model *jsql.Model
* @return string, error
**/
func (s *Sqlite) Load(model *jsql.Model) (string, error) {
	var sb strings.Builder

	table := ddlTable(model)
	cols := ddlColumns(model)
	pk := ddlPrimaryKey(model)
	fks := ddlForeignKeys(model)

	body := append([]string{}, cols...)
	if pk != "" {
		body = append(body, pk)
	}
	body = append(body, fks...)

	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table))
	sb.WriteString(strings.Join(body, ",\n"))
	sb.WriteString("\n);\n")

	for _, stmt := range ddlUnique(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
	}

	for _, stmt := range ddlIndexes(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
	}

	return sb.String(), nil
}
