package sqlite

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/jsql"
)

/**
* sqTable: Returns the table name for SQLite (no schema support).
* @param model *jsql.Model
* @return string
**/
func sqTable(model *jsql.Model) string {
	model.Table = model.Name
	return model.Name
}

/**
* sqDDLColumns: Builds the column definition list for CREATE TABLE.
* _source is emitted as TEXT DEFAULT '{}' (SQLite stores JSON as text).
* @param model *jsql.Model
* @return []string
**/
func sqDDLColumns(model *jsql.Model) []string {
	var cols []string
	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN {
			continue
		}
		if col.Name == model.SourceField {
			cols = append(cols, fmt.Sprintf("  %s TEXT DEFAULT '{}'", model.SourceField))
			continue
		}
		tp := sqType(col.TypeData)
		def := sqDefault(col.TypeData, col.Default)
		cols = append(cols, fmt.Sprintf("  %s %s DEFAULT %s", col.Name, tp, def))
	}
	return cols
}

/**
* sqDDLPrimaryKey: Builds the PRIMARY KEY table constraint clause, or empty string.
* SQLite requires PKs inside CREATE TABLE, not via ALTER TABLE.
* @param model *jsql.Model
* @return string
**/
func sqDDLPrimaryKey(model *jsql.Model) string {
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
* sqDDLForeignKeys: Builds FOREIGN KEY table constraint clauses for CREATE TABLE.
* SQLite FK constraints must be inside CREATE TABLE; ALTER TABLE ADD CONSTRAINT is not supported.
* @param model *jsql.Model
* @return []string
**/
func sqDDLForeignKeys(model *jsql.Model) []string {
	stmts := make([]string, 0, len(model.ForeignKeys))
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

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("  FOREIGN KEY (%s)\n", strings.Join(localCols, ", ")))
		sb.WriteString(fmt.Sprintf("    REFERENCES %s (%s)", foreignTable, strings.Join(foreignCols, ", ")))
		if fk.OnDeleteCascade {
			sb.WriteString("\n    ON DELETE CASCADE")
		}
		if fk.OnUpdateCascade {
			sb.WriteString("\n    ON UPDATE CASCADE")
		}
		stmts = append(stmts, sb.String())
	}
	return stmts
}

/**
* sqDDLUnique: Builds CREATE UNIQUE INDEX statements.
* @param model *jsql.Model
* @param table string
* @return []string
**/
func sqDDLUnique(model *jsql.Model, table string) []string {
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
* sqDDLIndexes: Builds CREATE INDEX statements.
* SQLite uses B-tree for all indexes — USING clause is not supported.
* @param model *jsql.Model
* @param table string
* @return []string
**/
func sqDDLIndexes(model *jsql.Model, table string) []string {
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
* Load: Generates the DDL SQL to create the table with primary key and foreign key constraints
* inline, followed by unique and regular index statements.
* @param model *jsql.Model
* @return string, error
**/
func (s *Sqlite) Load(model *jsql.Model) (string, error) {
	var sb strings.Builder

	table := sqTable(model)
	cols := sqDDLColumns(model)

	if pk := sqDDLPrimaryKey(model); pk != "" {
		cols = append(cols, pk)
	}
	for _, fk := range sqDDLForeignKeys(model) {
		cols = append(cols, fk)
	}

	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table))
	sb.WriteString(strings.Join(cols, ",\n"))
	sb.WriteString("\n);\n")

	for _, stmt := range sqDDLUnique(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	for _, stmt := range sqDDLIndexes(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
