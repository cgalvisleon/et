package postgres

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* pgColsVals: Separates data into sorted parallel (column names, quoted value strings) slices
* and a source et.Json for ATTRIB fields destined for the _source JSONB column.
* When excludePKs is true, primary key columns are omitted from the column lists (for SET clauses).
* @param model *jsql.Model
* @param data et.Json
* @param excludePKs bool
* @return []string, []string, et.Json
**/
func pgColsVals(model *jsql.Model, data et.Json, excludePKs bool) (cols, vals []string, source et.Json) {
	source = et.Json{}
	colMap := make(map[string]interface{})

	pkSet := make(map[string]bool, len(model.PrimaryKeys))
	if excludePKs {
		for _, pk := range model.PrimaryKeys {
			pkSet[pk.Name] = true
		}
	}

	for key, val := range data {
		if key == model.SourceField {
			continue
		}
		col, ok := model.GetColumn(key)
		if !ok {
			continue
		}
		switch col.TypeColumn {
		case jsql.COLUMN:
			if !pkSet[key] {
				colMap[key] = val
			}
		case jsql.ATTRIB:
			source[key] = val
		}
	}

	names := make([]string, 0, len(colMap))
	for k := range colMap {
		names = append(names, k)
	}
	sort.Strings(names)

	cols = make([]string, len(names))
	vals = make([]string, len(names))
	for i, name := range names {
		cols[i] = name
		vals[i] = fmt.Sprintf("%v", jsql.Quoted(colMap[name]))
	}
	return
}

/**
* pgAttribReturn: Builds a _source JSONB extraction expression for RETURNING clauses (no table alias).
* @param sourceField string
* @param field string
* @param tp jsql.TypeData
* @return string
**/
func pgAttribReturn(sourceField, field string, tp jsql.TypeData) string {
	path := fmt.Sprintf("%s->>'%s'", sourceField, field)
	switch tp {
	case jsql.INT:
		return fmt.Sprintf("(%s)::bigint AS %s", path, field)
	case jsql.FLOAT:
		return fmt.Sprintf("(%s)::double precision AS %s", path, field)
	case jsql.BOOLEAN:
		return fmt.Sprintf("(%s)::boolean AS %s", path, field)
	case jsql.DATETIME:
		return fmt.Sprintf("(%s)::timestamptz AS %s", path, field)
	default:
		return fmt.Sprintf("%s AS %s", path, field)
	}
}

/**
* pgReturningClause: Builds the RETURNING clause, expanding ATTRIB columns from _source inline.
* Uses command.Returns when set; otherwise auto-builds from the model column list.
* @param command *jsql.Command
* @return string
**/
func pgReturningClause(command *jsql.Command) string {
	if len(command.Returns) > 0 {
		return "\nRETURNING " + strings.Join(command.Returns, ", ")
	}

	model := command.From.Model
	if model == nil {
		return "\nRETURNING *"
	}

	exprs := make([]string, 0, len(model.Columns))
	for _, col := range model.Columns {
		switch col.TypeColumn {
		case jsql.COLUMN:
			if col.Name == model.SourceField {
				continue
			}
			exprs = append(exprs, col.Name)
		case jsql.ATTRIB:
			if model.SourceField == "" {
				continue
			}
			exprs = append(exprs, pgAttribReturn(model.SourceField, col.Name, col.TypeData))
		}
	}

	if len(exprs) == 0 {
		return "\nRETURNING *"
	}
	return "\nRETURNING " + strings.Join(exprs, ", ")
}

/**
* pgPKWhere: Builds a WHERE clause using primary key values from data.
* @param model *jsql.Model
* @param data et.Json
* @return string
**/
func pgPKWhere(model *jsql.Model, data et.Json) string {
	conds := make([]string, 0, len(model.PrimaryKeys))
	for _, pk := range model.PrimaryKeys {
		val, ok := data[pk.Name]
		if !ok {
			continue
		}
		conds = append(conds, fmt.Sprintf("%s = %v", pk.Name, jsql.Quoted(val)))
	}
	return strings.Join(conds, " AND ")
}

/**
* pgInsertSQL: Generates INSERT INTO … (cols) VALUES (vals) RETURNING …
* @param command *jsql.Command
* @return string, error
**/
func pgInsertSQL(command *jsql.Command) (string, error) {
	table := pgFromRef(command.From)
	model := command.From.Model

	var cols, vals []string
	var source et.Json

	if model != nil {
		cols, vals, source = pgColsVals(model, command.New, false)
		if model.SourceField != "" && len(source) > 0 {
			cols = append(cols, model.SourceField)
			vals = append(vals, fmt.Sprintf("%v::jsonb", jsql.Quoted(source)))
		}
	} else {
		keys := make([]string, 0, len(command.New))
		for k := range command.New {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		cols = make([]string, len(keys))
		vals = make([]string, len(keys))
		for i, k := range keys {
			cols[i] = k
			vals[i] = fmt.Sprintf("%v", jsql.Quoted(command.New[k]))
		}
	}

	if len(cols) == 0 {
		return "", fmt.Errorf("no columns to insert into %s", table)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO %s\n", table))
	sb.WriteString(fmt.Sprintf("  (%s)\n", strings.Join(cols, ", ")))
	sb.WriteString(fmt.Sprintf("VALUES\n  (%s)", strings.Join(vals, ", ")))
	sb.WriteString(pgReturningClause(command))
	sb.WriteString(";")
	return sb.String(), nil
}

/**
* pgUpdateSQL: Generates UPDATE … SET … WHERE … RETURNING …
* Excludes primary key columns from SET; WHERE uses PK values from command.New.
* @param command *jsql.Command
* @return string, error
**/
func pgUpdateSQL(command *jsql.Command) (string, error) {
	table := pgFromRef(command.From)
	model := command.From.Model

	var setCols []string

	if model != nil {
		cols, vals, source := pgColsVals(model, command.New, true)
		for i, col := range cols {
			setCols = append(setCols, fmt.Sprintf("%s = %s", col, vals[i]))
		}
		if model.SourceField != "" && len(source) > 0 {
			setCols = append(setCols, fmt.Sprintf("%s = %v::jsonb", model.SourceField, jsql.Quoted(source)))
		}
	} else {
		keys := make([]string, 0, len(command.New))
		for k := range command.New {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			setCols = append(setCols, fmt.Sprintf("%s = %v", k, jsql.Quoted(command.New[k])))
		}
	}

	if len(setCols) == 0 {
		return "", fmt.Errorf("no columns to update in %s", table)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("UPDATE %s\n", table))
	sb.WriteString("SET\n  " + strings.Join(setCols, ",\n  "))

	var whereSQL string
	if model != nil && len(model.PrimaryKeys) > 0 {
		whereSQL = pgPKWhere(model, command.New)
	}
	if whereSQL == "" && len(command.Conditions) > 0 {
		whereSQL = pgCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL != "" {
		sb.WriteString("\nWHERE " + whereSQL)
	}

	sb.WriteString(pgReturningClause(command))
	sb.WriteString(";")
	return sb.String(), nil
}

/**
* pgDeleteSQL: Generates DELETE FROM … WHERE … RETURNING …
* WHERE uses primary key values from command.Old (the fetched row).
* @param command *jsql.Command
* @return string, error
**/
func pgDeleteSQL(command *jsql.Command) (string, error) {
	table := pgFromRef(command.From)
	model := command.From.Model

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("DELETE FROM %s", table))

	var whereSQL string
	if model != nil && len(model.PrimaryKeys) > 0 && len(command.Old) > 0 {
		whereSQL = pgPKWhere(model, command.Old)
	}
	if whereSQL == "" && len(command.Conditions) > 0 {
		whereSQL = pgCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL != "" {
		sb.WriteString("\nWHERE " + whereSQL)
	}

	sb.WriteString(pgReturningClause(command))
	sb.WriteString(";")
	return sb.String(), nil
}

/**
* Command: Generates the SQL DML string (INSERT, UPDATE, DELETE, BULK) for the given Command.
* @param command *jsql.Command
* @return string, error
**/
func (s *Postgres) Command(command *jsql.Command) (string, error) {
	switch command.Type {
	case jsql.INSERT, jsql.BULK:
		return pgInsertSQL(command)
	case jsql.UPDATE:
		return pgUpdateSQL(command)
	case jsql.DELETE:
		return pgDeleteSQL(command)
	default:
		return "", fmt.Errorf("unsupported command type: %s", command.Type)
	}
}
