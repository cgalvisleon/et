package postgres

import (
	"database/sql"
	"fmt"
	"sort"
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
		model.Table = fmt.Sprintf("%s.%s", model.Schema, model.Name)
	} else {
		model.Table = fmt.Sprintf("%s", model.Name)
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

	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN {
			continue
		}
		if col.Name == model.SourceField {
			cols = append(cols, fmt.Sprintf("  %s JSONB DEFAULT '{}'", model.SourceField))
			continue
		}
		tp := pgType(col.TypeData)
		def := pgDefault(col.TypeData, col.Default)
		line := fmt.Sprintf("  %s %s DEFAULT %s", col.Name, tp, def)
		cols = append(cols, line)
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
* ddlForeignKeys: Builds ALTER TABLE … ADD CONSTRAINT … FOREIGN KEY statements for each FK.
* Keys map entries are sorted for deterministic output.
* ON DELETE / ON UPDATE CASCADE clauses are added when the respective flag is set.
* @param model *jsql.Model
* @param table string
* @return []string
**/
func ddlForeignKeys(model *jsql.Model, table string) []string {
	base := strings.ReplaceAll(table, ".", "_")
	stmts := make([]string, 0, len(model.ForeignKeys))
	for _, fk := range model.ForeignKeys {
		if fk.To == nil || len(fk.Keys) == 0 {
			continue
		}

		foreignTable := fk.To.Name
		if fk.To.Schema != "" {
			foreignTable = fmt.Sprintf("%s.%s", fk.To.Schema, fk.To.Name)
		}
		foreignBase := strings.ReplaceAll(foreignTable, ".", "_")

		localCols := make([]string, 0, len(fk.Keys))
		for local := range fk.Keys {
			localCols = append(localCols, local)
		}
		sort.Strings(localCols)

		foreignCols := make([]string, len(localCols))
		for i, local := range localCols {
			foreignCols[i] = fk.Keys[local]
		}

		constraintName := fmt.Sprintf("fk_%s_%s", base, foreignBase)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s\n", table, constraintName))
		sb.WriteString(fmt.Sprintf("  FOREIGN KEY (%s)\n", strings.Join(localCols, ", ")))
		sb.WriteString(fmt.Sprintf("  REFERENCES %s (%s)", foreignTable, strings.Join(foreignCols, ", ")))
		if fk.OnDeleteCascade {
			sb.WriteString("\n  ON DELETE CASCADE")
		}
		if fk.OnUpdateCascade {
			sb.WriteString("\n  ON UPDATE CASCADE")
		}
		sb.WriteString(";")
		stmts = append(stmts, sb.String())
	}
	return stmts
}

/**
* ExistModel: Returns true when a table with the given schema and name exists in the database.
* @param db *sql.DB @param schema string @param name string
* @return bool, error
**/
func (s *Postgres) ExistModel(db *sql.DB, schema, name string) (bool, error) {
	query := `
	SELECT EXISTS(
	SELECT 1
	FROM information_schema.tables
	WHERE UPPER(table_schema) = UPPER($1)
	AND UPPER(table_name) = UPPER($2));`
	rows, err := db.Query(query, schema, name)
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
* Load: Generates the DDL SQL to create the schema, table, primary key,
* unique indexes, regular indexes, and foreign key constraints for the given model.
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
	}

	for _, stmt := range ddlUnique(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
	}

	for _, stmt := range ddlIndexes(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
	}

	for _, stmt := range ddlForeignKeys(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
	}

	return sb.String(), nil
}
