package postgres

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/jsql"
)

/**
* ddlSchema: Emits CREATE SCHEMA IF NOT EXISTS for the model's schema.
* @param model *jsql.Model
* @return string
**/
func ddlSchema(model *jsql.Model) string {
	if model.Schema == "" {
		return ""
	}
	return fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", model.Schema)
}

/**
* ddlTable: Builds the full qualified table identifier.
* @param model *jsql.Model
* @return string
**/
func ddlTable(model *jsql.Model) string {
	if model.Schema != "" {
		return fmt.Sprintf("%s.%s", model.Schema, model.Table)
	}
	return model.Table
}

/**
* ddlColumns: Builds the column definition list for CREATE TABLE.
* Emits:
*   - _idx BIGSERIAL             (when IdxField is set)
*   - real PG columns for COLUMN type
*   - _source JSONB DEFAULT '{}'  (when SourceField is set)
* @param model *jsql.Model
* @return []string
**/
func ddlColumns(model *jsql.Model) []string {
	var cols []string

	if model.IdxField != "" {
		cols = append(cols, fmt.Sprintf("  %s VARCHAR(80) DEFAULT ''", model.IdxField))
	}

	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN {
			continue
		}
		tp := pgType(col.TypeData)
		def := pgDefault(col.TypeData, col.Default)
		line := fmt.Sprintf("  %s %s DEFAULT %s", col.Name, tp, def)
		cols = append(cols, line)
	}

	if model.SourceField != "" {
		cols = append(cols, fmt.Sprintf("  %s JSONB DEFAULT '{}'", model.SourceField))
	}

	return cols
}

/**
* ddlPrimaryKey: Builds the PRIMARY KEY constraint clause, or empty string.
* @param model *jsql.Model
* @param table string
* @return string
**/
func ddlPrimaryKey(model *jsql.Model, table string) string {
	if len(model.PrimaryKeys) == 0 {
		return ""
	}
	keys := make([]string, len(model.PrimaryKeys))
	for i, k := range model.PrimaryKeys {
		keys[i] = k.Name
	}
	constraintName := strings.ReplaceAll(table, ".", "_") + "_pkey"
	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (%s);",
		table, constraintName, strings.Join(keys, ", "))
}

/**
* ddlUnique: Builds UNIQUE INDEX statements.
* @param model *jsql.Model
* @param table string
* @return []string
**/
func ddlUnique(model *jsql.Model, table string) []string {
	base := strings.ReplaceAll(table, ".", "_")
	stmts := make([]string, 0, len(model.Unique))
	for _, u := range model.Unique {
		idxName := fmt.Sprintf("%s_%s_key", base, u.Name)
		stmts = append(stmts, fmt.Sprintf(
			"CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s);",
			idxName, table, u.Name))
	}
	return stmts
}

/**
* ddlIndexes: Builds CREATE INDEX statements for regular indexes.
* @param model *jsql.Model
* @param table string
* @return []string
**/
func ddlIndexes(model *jsql.Model, table string) []string {
	base := strings.ReplaceAll(table, ".", "_")
	stmts := make([]string, 0, len(model.Indexes))
	for _, idx := range model.Indexes {
		idxName := fmt.Sprintf("%s_%s_idx", base, idx.Name)
		method := "HASH"
		if idx.Sorted {
			method = "BTREE"
		}
		stmts = append(stmts, fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s ON %s USING %s (%s);",
			idxName, table, method, idx.Name))
	}
	return stmts
}

/**
* Load: Generates the DDL SQL to create the schema, table, primary key,
* unique indexes, and regular indexes for the given model.
* Returns the complete DDL as a single string with statements separated by newlines.
* @param model *jsql.Model
* @return string, error
**/
func (s *Postgres) Load(model *jsql.Model) (string, error) {
	var sb strings.Builder

	if schema := ddlSchema(model); schema != "" {
		sb.WriteString(schema)
		sb.WriteString("\n\n")
	}

	table := ddlTable(model)
	cols := ddlColumns(model)

	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table))
	sb.WriteString(strings.Join(cols, ",\n"))
	sb.WriteString("\n);\n")

	if pk := ddlPrimaryKey(model, table); pk != "" {
		sb.WriteString("\n")
		sb.WriteString(pk)
		sb.WriteString("\n")
	}

	for _, stmt := range ddlUnique(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	for _, stmt := range ddlIndexes(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
