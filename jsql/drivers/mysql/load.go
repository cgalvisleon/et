package mysql

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/jsql"
)

/**
* myTable: Returns the table name for MySQL (no schema — uses current database).
* @param model *jsql.Model
* @return string
**/
func myTable(model *jsql.Model) string {
	model.Table = model.Name
	return model.Name
}

/**
* myDDLColumns: Builds the column definition list for CREATE TABLE.
* _source is emitted as JSON DEFAULT (JSON_OBJECT()) — requires MySQL 8.0.13+.
* @param model *jsql.Model
* @return []string
**/
func myDDLColumns(model *jsql.Model) []string {
	var cols []string
	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN {
			continue
		}
		if col.Name == model.SourceField {
			cols = append(cols, fmt.Sprintf("  `%s` JSON DEFAULT (JSON_OBJECT())", model.SourceField))
			continue
		}
		tp := myType(col.TypeData)
		def := myDefault(col.TypeData, col.Default)
		cols = append(cols, fmt.Sprintf("  `%s` %s DEFAULT %s", col.Name, tp, def))
	}
	return cols
}

/**
* myDDLPrimaryKey: Builds the PRIMARY KEY table constraint clause, or empty string.
* @param model *jsql.Model
* @return string
**/
func myDDLPrimaryKey(model *jsql.Model) string {
	if len(model.PrimaryKeys) == 0 {
		return ""
	}
	keys := make([]string, len(model.PrimaryKeys))
	for i, k := range model.PrimaryKeys {
		keys[i] = fmt.Sprintf("`%s`", k.Name)
	}
	return fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(keys, ", "))
}

/**
* myDDLUnique: Builds CREATE UNIQUE INDEX statements.
* @param model *jsql.Model
* @param table string
* @return []string
**/
func myDDLUnique(model *jsql.Model, table string) []string {
	stmts := make([]string, 0, len(model.Unique))
	for _, u := range model.Unique {
		idxName := fmt.Sprintf("%s_%s_key", table, u.Name)
		stmts = append(stmts, fmt.Sprintf(
			"CREATE UNIQUE INDEX `%s` ON `%s` (`%s`);",
			idxName, table, u.Name))
	}
	return stmts
}

/**
* myDDLIndexes: Builds CREATE INDEX statements.
* MySQL InnoDB uses B-tree for all indexes — USING clause is optional and defaults to BTREE.
* @param model *jsql.Model
* @param table string
* @return []string
**/
func myDDLIndexes(model *jsql.Model, table string) []string {
	stmts := make([]string, 0, len(model.Indexes))
	for _, idx := range model.Indexes {
		idxName := fmt.Sprintf("%s_%s_idx", table, idx.Name)
		stmts = append(stmts, fmt.Sprintf(
			"CREATE INDEX `%s` ON `%s` (`%s`);",
			idxName, table, idx.Name))
	}
	return stmts
}

/**
* myDDLForeignKeys: Builds ALTER TABLE … ADD CONSTRAINT … FOREIGN KEY statements.
* MySQL InnoDB supports FK constraints via ALTER TABLE (unlike SQLite).
* @param model *jsql.Model
* @param table string
* @return []string
**/
func myDDLForeignKeys(model *jsql.Model, table string) []string {
	base := strings.ReplaceAll(table, ".", "_")
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

		quotedLocal := make([]string, len(localCols))
		quotedForeign := make([]string, len(foreignCols))
		for i, c := range localCols {
			quotedLocal[i] = fmt.Sprintf("`%s`", c)
		}
		for i, c := range foreignCols {
			quotedForeign[i] = fmt.Sprintf("`%s`", c)
		}

		constraintName := fmt.Sprintf("fk_%s_%s", base, strings.ReplaceAll(foreignTable, ".", "_"))

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s`\n", table, constraintName))
		sb.WriteString(fmt.Sprintf("  FOREIGN KEY (%s)\n", strings.Join(quotedLocal, ", ")))
		sb.WriteString(fmt.Sprintf("  REFERENCES `%s` (%s)", foreignTable, strings.Join(quotedForeign, ", ")))
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
* Load: Generates the DDL SQL to create the table (InnoDB, utf8mb4), primary key,
* unique indexes, regular indexes, and foreign key constraints for MySQL.
* Requires MySQL 8.0.13+ for JSON DEFAULT (JSON_OBJECT()).
* @param model *jsql.Model
* @return string, error
**/
func (s *MySQL) Load(model *jsql.Model) (string, error) {
	var sb strings.Builder

	table := myTable(model)
	cols := myDDLColumns(model)

	if pk := myDDLPrimaryKey(model); pk != "" {
		cols = append(cols, pk)
	}

	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n", table))
	sb.WriteString(strings.Join(cols, ",\n"))
	sb.WriteString("\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\n")

	for _, stmt := range myDDLUnique(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	for _, stmt := range myDDLIndexes(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	for _, stmt := range myDDLForeignKeys(model, table) {
		sb.WriteString("\n")
		sb.WriteString(stmt)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
