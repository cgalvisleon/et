package oracle

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
* ddlTable: Sets model.Table (schema.name, as the other drivers do) and returns the quoted table reference.
* @param model *jsql.Model
* @return string
**/
func ddlTable(model *jsql.Model) string {
	if model.Schema != "" {
		model.Table = fmt.Sprintf("%s.%s", model.Schema, model.Name)
	} else {
		model.Table = model.Name
	}
	return oraTableRef(model.Schema, model.Name)
}

/**
* ddlColumns: Builds the column definition list for CREATE TABLE (only COLUMN types).
* JSON columns, including the SourceField, get an IS JSON check constraint.
* @param model *jsql.Model
* @return []string
**/
func ddlColumns(model *jsql.Model) []string {
	var cols []string
	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN {
			continue
		}
		name := oraIdent(col.Name)
		if col.Name == model.SourceField {
			cols = append(cols, fmt.Sprintf("  %s CLOB DEFAULT TO_CLOB('{}') CHECK (%s IS JSON)", name, name))
			continue
		}
		line := fmt.Sprintf("  %s %s DEFAULT %s", name, oraType(col.TypeData), oraDefault(col.TypeData, col.Default))
		if isJsonType(col.TypeData) {
			line += fmt.Sprintf(" CHECK (%s IS JSON)", name)
		}
		cols = append(cols, line)
	}
	return cols
}

/**
* columnType: Returns the TypeData of a model column, or et.ANY when it is not a real column.
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
* ddlIndexes: Builds PRIMARY KEY, UNIQUE INDEX and INDEX statements. Oracle rejects a second index
* on an already indexed column and cannot index LOBs, so those are skipped.
* @param model *jsql.Model, table string
* @return []string
**/
func ddlIndexes(model *jsql.Model, table string) []string {
	base := strings.ReplaceAll(model.Table, ".", "_")
	stmts := make([]string, 0)
	indexed := make([]string, 0)

	if len(model.PrimaryKeys) > 0 {
		keys := make([]string, len(model.PrimaryKeys))
		for i, k := range model.PrimaryKeys {
			keys[i] = oraIdent(k.Name)
		}
		stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (%s)",
			table, oraObjectName(base, "pkey"), strings.Join(keys, ", ")))
		if len(keys) == 1 {
			indexed = append(indexed, model.PrimaryKeys[0].Name)
		}
	}

	for _, u := range model.Unique {
		if slices.Contains(indexed, u.Name) || isLobType(columnType(model, u.Name)) {
			continue
		}
		stmts = append(stmts, fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s)",
			oraObjectName(base, u.Name, "key"), table, oraIdent(u.Name)))
		indexed = append(indexed, u.Name)
	}

	for _, idx := range model.Indexes {
		if slices.Contains(indexed, idx.Name) || isLobType(columnType(model, idx.Name)) {
			continue
		}
		stmts = append(stmts, fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
			oraObjectName(base, idx.Name, "idx"), table, oraIdent(idx.Name)))
		indexed = append(indexed, idx.Name)
	}

	return stmts
}

/**
* ddlForeignKeys: Builds ALTER TABLE … ADD CONSTRAINT … FOREIGN KEY statements for each FK.
* Oracle has no ON UPDATE CASCADE, so only ON DELETE CASCADE is emitted.
* @param model *jsql.Model, table string
* @return []string
**/
func ddlForeignKeys(model *jsql.Model, table string) []string {
	base := strings.ReplaceAll(model.Table, ".", "_")
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
			locals[i] = oraIdent(local)
			foreigns[i] = oraIdent(fk.Keys[local])
		}

		foreignBase := strings.ReplaceAll(fk.To.Schema+"_"+fk.To.Name, ".", "_")
		stmt := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
			table, oraObjectName("fk", base, foreignBase), strings.Join(locals, ", "),
			oraTableRef(fk.To.Schema, fk.To.Name), strings.Join(foreigns, ", "))
		if fk.OnDeleteCascade {
			stmt += " ON DELETE CASCADE"
		}
		stmts = append(stmts, stmt)
	}
	return stmts
}

/**
* ExistModel: Returns true when the model's table exists in its schema (or in the current schema).
* @param db *sql.DB, model *jsql.Model
* @return bool, error
**/
func (s *Oracle) ExistModel(db *sql.DB, model *jsql.Model) (bool, error) {
	ddlTable(model)
	owner := "SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA')"
	if model.Schema != "" {
		owner = fmt.Sprintf("'%s'", oraSchema(model.Schema))
	}
	query := fmt.Sprintf(`SELECT CASE WHEN EXISTS(
	SELECT 1 FROM ALL_TABLES
	WHERE OWNER = %s
	AND TABLE_NAME = '%s') THEN 'true' ELSE 'false' END AS "exists" FROM DUAL`,
		owner, jsql.EscapeSQLString(sanitizeIdent(model.Name)))
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
* Load: Generates the DDL for the given model as one PL/SQL block (go-ora runs one statement per call):
* CREATE TABLE, primary key, unique indexes, indexes and foreign keys, each through EXECUTE IMMEDIATE.
* The schema is not created: in Oracle it is a user and must already exist.
* @param model *jsql.Model
* @return string, error
**/
func (s *Oracle) Load(model *jsql.Model) (string, error) {
	table := ddlTable(model)
	cols := ddlColumns(model)
	if len(cols) == 0 {
		return "", fmt.Errorf("model %s has no columns", model.Table)
	}

	stmts := []string{fmt.Sprintf("CREATE TABLE %s (\n%s\n)", table, strings.Join(cols, ",\n"))}
	stmts = append(stmts, ddlIndexes(model, table)...)
	stmts = append(stmts, ddlForeignKeys(model, table)...)

	var sb strings.Builder
	sb.WriteString("BEGIN\n")
	for _, stmt := range stmts {
		sb.WriteString(fmt.Sprintf("  EXECUTE IMMEDIATE '%s';\n", jsql.EscapeSQLString(stmt)))
	}
	sb.WriteString("END;")
	return sb.String(), nil
}
