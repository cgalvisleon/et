package mssql

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/jsql"
)

/**
* msFromRef: Returns the fully-qualified [schema].[name] table identifier.
* Falls back to just [name] when schema is empty.
* @param model *jsql.Model
* @return string
**/
func msTableRef(model *jsql.Model) string {
	if model.Schema != "" {
		model.Table = fmt.Sprintf("[%s].[%s]", model.Schema, model.Name)
	} else {
		model.Table = fmt.Sprintf("[%s]", model.Name)
	}
	return model.Table
}

/**
* msDDLSchema: Emits a conditional CREATE SCHEMA block.
* SQL Server requires EXEC for CREATE SCHEMA inside a batch.
* @param model *jsql.Model
* @return string
**/
func msDDLSchema(model *jsql.Model) string {
	if model.Schema == "" {
		return ""
	}
	return fmt.Sprintf(
		"IF NOT EXISTS (SELECT 1 FROM sys.schemas WHERE name = N'%s')\n    EXEC(N'CREATE SCHEMA [%s]');",
		model.Schema, model.Schema)
}

/**
* msDDLColumns: Builds the column definition list for CREATE TABLE.
* _source is emitted as NVARCHAR(MAX) DEFAULT N'{}' (JSON stored as Unicode text).
* @param model *jsql.Model
* @return []string
**/
func msDDLColumns(model *jsql.Model) []string {
	var cols []string
	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN {
			continue
		}
		if col.Name == model.SourceField {
			cols = append(cols, fmt.Sprintf("    [%s] NVARCHAR(MAX) DEFAULT N'{}'", model.SourceField))
			continue
		}
		tp := msType(col.TypeData)
		def := msDefault(col.TypeData, col.Default)
		cols = append(cols, fmt.Sprintf("    [%s] %s DEFAULT %s", col.Name, tp, def))
	}
	return cols
}

/**
* msDDLPrimaryKey: Builds a named PRIMARY KEY constraint clause for CREATE TABLE, or empty string.
* @param model *jsql.Model
* @param table string
* @return string
**/
func msDDLPrimaryKey(model *jsql.Model, table string) string {
	if len(model.PrimaryKeys) == 0 {
		return ""
	}
	keys := make([]string, len(model.PrimaryKeys))
	for i, k := range model.PrimaryKeys {
		keys[i] = fmt.Sprintf("[%s]", k.Name)
	}
	constraintName := strings.ReplaceAll(strings.Trim(table, "[]"), "].[", "_") + "_pkey"
	return fmt.Sprintf("    CONSTRAINT [%s] PRIMARY KEY (%s)", constraintName, strings.Join(keys, ", "))
}

/**
* msDDLUnique: Builds conditional CREATE UNIQUE INDEX statements.
* @param model *jsql.Model
* @param table string
* @return []string
**/
func msDDLUnique(model *jsql.Model, table string) []string {
	base := strings.ReplaceAll(strings.Trim(table, "[]"), "].[", "_")
	stmts := make([]string, 0, len(model.Unique))
	for _, u := range model.Unique {
		idxName := fmt.Sprintf("%s_%s_key", base, u.Name)
		stmts = append(stmts, fmt.Sprintf(
			"IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'%s' AND object_id = OBJECT_ID(N'%s'))\n"+
				"    CREATE UNIQUE INDEX [%s] ON %s ([%s]);",
			idxName, table, idxName, table, u.Name))
	}
	return stmts
}

/**
* msDDLIndexes: Builds conditional CREATE INDEX statements (NONCLUSTERED B-tree by default).
* @param model *jsql.Model
* @param table string
* @return []string
**/
func msDDLIndexes(model *jsql.Model, table string) []string {
	base := strings.ReplaceAll(strings.Trim(table, "[]"), "].[", "_")
	stmts := make([]string, 0, len(model.Indexes))
	for _, idx := range model.Indexes {
		idxName := fmt.Sprintf("%s_%s_idx", base, idx.Name)
		stmts = append(stmts, fmt.Sprintf(
			"IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'%s' AND object_id = OBJECT_ID(N'%s'))\n"+
				"    CREATE INDEX [%s] ON %s ([%s]);",
			idxName, table, idxName, table, idx.Name))
	}
	return stmts
}

/**
* msDDLForeignKeys: Builds conditional ALTER TABLE … ADD CONSTRAINT … FOREIGN KEY statements.
* @param model *jsql.Model
* @param table string
* @return []string
**/
func msDDLForeignKeys(model *jsql.Model, table string) []string {
	base := strings.ReplaceAll(strings.Trim(table, "[]"), "].[", "_")
	stmts := make([]string, 0, len(model.ForeignKeys))
	for _, fk := range model.ForeignKeys {
		if fk.To == nil || len(fk.Keys) == 0 {
			continue
		}

		var foreignTable string
		if fk.To.Schema != "" {
			foreignTable = fmt.Sprintf("[%s].[%s]", fk.To.Schema, fk.To.Name)
		} else {
			foreignTable = fmt.Sprintf("[%s]", fk.To.Name)
		}
		foreignBase := strings.ReplaceAll(strings.Trim(foreignTable, "[]"), "].[", "_")

		localCols := make([]string, 0, len(fk.Keys))
		for local := range fk.Keys {
			localCols = append(localCols, local)
		}
		sort.Strings(localCols)

		foreignCols := make([]string, len(localCols))
		for i, local := range localCols {
			foreignCols[i] = fk.Keys[local]
		}

		quotedLocal := make([]string, len(localCols))
		quotedForeign := make([]string, len(foreignCols))
		for i, c := range localCols {
			quotedLocal[i] = fmt.Sprintf("[%s]", c)
		}
		for i, c := range foreignCols {
			quotedForeign[i] = fmt.Sprintf("[%s]", c)
		}

		constraintName := fmt.Sprintf("fk_%s_%s", base, foreignBase)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("IF NOT EXISTS (SELECT 1 FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS WHERE CONSTRAINT_NAME = N'%s')\n", constraintName))
		sb.WriteString(fmt.Sprintf("    ALTER TABLE %s ADD CONSTRAINT [%s]\n", table, constraintName))
		sb.WriteString(fmt.Sprintf("        FOREIGN KEY (%s)\n", strings.Join(quotedLocal, ", ")))
		sb.WriteString(fmt.Sprintf("        REFERENCES %s (%s)", foreignTable, strings.Join(quotedForeign, ", ")))
		if fk.OnDeleteCascade {
			sb.WriteString("\n        ON DELETE CASCADE")
		}
		if fk.OnUpdateCascade {
			sb.WriteString("\n        ON UPDATE CASCADE")
		}
		sb.WriteString(";")
		stmts = append(stmts, sb.String())
	}
	return stmts
}

/**
* Load: Generates T-SQL DDL to create the schema, table, primary key, unique indexes,
* regular indexes, and foreign key constraints. All statements use IF NOT EXISTS guards.
* @param model *jsql.Model
* @return string, error
**/
func (s *MSSQL) Load(model *jsql.Model) (string, error) {
	var sb strings.Builder

	if schema := msDDLSchema(model); schema != "" {
		sb.WriteString(schema)
		sb.WriteString("\n\n")
	}

	table := msTableRef(model)
	cols := msDDLColumns(model)

	if pk := msDDLPrimaryKey(model, table); pk != "" {
		cols = append(cols, pk)
	}

	sb.WriteString(fmt.Sprintf("IF OBJECT_ID(N'%s', N'U') IS NULL\n", table))
	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table))
	sb.WriteString(strings.Join(cols, ",\n"))
	sb.WriteString("\n);\n")

	for _, stmt := range msDDLUnique(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	for _, stmt := range msDDLIndexes(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	for _, stmt := range msDDLForeignKeys(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
