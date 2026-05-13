package oracle

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/jsql"
)

/**
* orTableRef: Returns the fully-qualified "schema"."name" table identifier.
* Falls back to just "name" when schema is empty.
* @param model *jsql.Model
* @return string
**/
func orTableRef(model *jsql.Model) string {
	if model.Schema != "" {
		model.Table = fmt.Sprintf(`"%s"."%s"`, model.Schema, model.Name)
	} else {
		model.Table = fmt.Sprintf(`"%s"`, model.Name)
	}
	return model.Table
}

/**
* orDDLColumns: Builds the column definition list for CREATE TABLE.
* _source is emitted as CLOB DEFAULT '{}' (JSON stored as CLOB in Oracle 19c+).
* @param model *jsql.Model
* @return []string
**/
func orDDLColumns(model *jsql.Model) []string {
	var cols []string
	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN {
			continue
		}
		if col.Name == model.SourceField {
			cols = append(cols, fmt.Sprintf(`    "%s" CLOB DEFAULT '{}'`, model.SourceField))
			continue
		}
		tp := orType(col.TypeData)
		def := orDefault(col.TypeData, col.Default)
		cols = append(cols, fmt.Sprintf(`    "%s" %s DEFAULT %s`, col.Name, tp, def))
	}
	return cols
}

/**
* orDDLPrimaryKey: Builds a named CONSTRAINT PRIMARY KEY clause for CREATE TABLE, or empty string.
* @param model *jsql.Model
* @param table string
* @return string
**/
func orDDLPrimaryKey(model *jsql.Model, table string) string {
	if len(model.PrimaryKeys) == 0 {
		return ""
	}
	keys := make([]string, len(model.PrimaryKeys))
	for i, k := range model.PrimaryKeys {
		keys[i] = fmt.Sprintf(`"%s"`, k.Name)
	}
	constraintName := strings.ReplaceAll(strings.Trim(table, `"`), `"."`, "_") + "_pkey"
	return fmt.Sprintf(`    CONSTRAINT "%s" PRIMARY KEY (%s)`, constraintName, strings.Join(keys, ", "))
}

/**
* orExecBlock: Wraps an Oracle DDL statement in a BEGIN/EXECUTE IMMEDIATE/EXCEPTION block.
* Uses q'|...|' quoting to avoid escaping single quotes inside the DDL text.
* ORA-00955 (-955) is silently ignored — it means the object already exists.
* @param ddl string
* @return string
**/
func orExecBlock(ddl string) string {
	return fmt.Sprintf(
		"  BEGIN\n    EXECUTE IMMEDIATE q'|%s|';\n  EXCEPTION WHEN OTHERS THEN\n    IF SQLCODE != -955 THEN RAISE; END IF;\n  END;",
		ddl)
}

/**
* orDDLUnique: Builds conditional CREATE UNIQUE INDEX statements wrapped in PL/SQL EXCEPTION blocks.
* @param model *jsql.Model
* @param table string
* @return []string
**/
func orDDLUnique(model *jsql.Model, table string) []string {
	base := strings.ReplaceAll(strings.Trim(table, `"`), `"."`, "_")
	stmts := make([]string, 0, len(model.Unique))
	for _, u := range model.Unique {
		idxName := fmt.Sprintf("%s_%s_key", base, u.Name)
		ddl := fmt.Sprintf(`CREATE UNIQUE INDEX "%s" ON %s ("%s")`, idxName, table, u.Name)
		stmts = append(stmts, orExecBlock(ddl))
	}
	return stmts
}

/**
* orDDLIndexes: Builds conditional CREATE INDEX statements wrapped in PL/SQL EXCEPTION blocks.
* Oracle uses B-tree by default; HASH index type is mapped to standard INDEX.
* @param model *jsql.Model
* @param table string
* @return []string
**/
func orDDLIndexes(model *jsql.Model, table string) []string {
	base := strings.ReplaceAll(strings.Trim(table, `"`), `"."`, "_")
	stmts := make([]string, 0, len(model.Indexes))
	for _, idx := range model.Indexes {
		idxName := fmt.Sprintf("%s_%s_idx", base, idx.Name)
		ddl := fmt.Sprintf(`CREATE INDEX "%s" ON %s ("%s")`, idxName, table, idx.Name)
		stmts = append(stmts, orExecBlock(ddl))
	}
	return stmts
}

/**
* orDDLForeignKeys: Builds conditional ALTER TABLE … ADD CONSTRAINT … FOREIGN KEY statements
* wrapped in PL/SQL EXCEPTION blocks. ORA-02275 (-2275) is also silenced (FK already exists).
* @param model *jsql.Model
* @param table string
* @return []string
**/
func orDDLForeignKeys(model *jsql.Model, table string) []string {
	base := strings.ReplaceAll(strings.Trim(table, `"`), `"."`, "_")
	stmts := make([]string, 0, len(model.ForeignKeys))
	for _, fk := range model.ForeignKeys {
		if fk.To == nil || len(fk.Keys) == 0 {
			continue
		}

		var foreignTable string
		if fk.To.Schema != "" {
			foreignTable = fmt.Sprintf(`"%s"."%s"`, fk.To.Schema, fk.To.Name)
		} else {
			foreignTable = fmt.Sprintf(`"%s"`, fk.To.Name)
		}
		foreignBase := strings.ReplaceAll(strings.Trim(foreignTable, `"`), `"."`, "_")

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
			quotedLocal[i] = fmt.Sprintf(`"%s"`, c)
		}
		for i, c := range foreignCols {
			quotedForeign[i] = fmt.Sprintf(`"%s"`, c)
		}

		constraintName := fmt.Sprintf("fk_%s_%s", base, foreignBase)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT "%s"`, table, constraintName))
		sb.WriteString(fmt.Sprintf("\n        FOREIGN KEY (%s)", strings.Join(quotedLocal, ", ")))
		sb.WriteString(fmt.Sprintf("\n        REFERENCES %s (%s)", foreignTable, strings.Join(quotedForeign, ", ")))
		if fk.OnDeleteCascade {
			sb.WriteString("\n        ON DELETE CASCADE")
		}

		// Oracle doesn't support ON UPDATE CASCADE; skip silently.
		stmts = append(stmts, orExecBlock(sb.String()))
	}
	return stmts
}

/**
* Load: Generates Oracle DDL as a single PL/SQL anonymous block containing EXECUTE IMMEDIATE
* calls wrapped in EXCEPTION handlers so repeated runs are idempotent (ORA-00955 suppressed).
* Oracle schemas equal database users and are assumed to exist; no CREATE SCHEMA is emitted.
* @param model *jsql.Model
* @return string, error
**/
func (s *Oracle) Load(model *jsql.Model) (string, error) {
	table := orTableRef(model)
	cols := orDDLColumns(model)

	if pk := orDDLPrimaryKey(model, table); pk != "" {
		cols = append(cols, pk)
	}

	createTable := fmt.Sprintf("CREATE TABLE %s (\n%s\n  )", table, strings.Join(cols, ",\n"))

	var sb strings.Builder
	sb.WriteString("BEGIN\n")
	sb.WriteString(orExecBlock(createTable))
	sb.WriteString("\n")

	for _, stmt := range orDDLUnique(model, table) {
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	for _, stmt := range orDDLIndexes(model, table) {
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	for _, stmt := range orDDLForeignKeys(model, table) {
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	sb.WriteString("END;")
	return sb.String(), nil
}
