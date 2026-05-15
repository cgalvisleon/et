package sqlite

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* sqJsonSet: Builds a json_set expression to patch individual ATTRIB keys into sourceField.
* SQLite's json_set accepts multiple path-value pairs in a single call and always creates missing keys.
* @param sourceField string, source et.Json
* @return string
**/
func sqJsonSet(sourceField string, source et.Json) string {
	keys := make([]string, 0, len(source))
	for k := range source {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	args := make([]string, 0, 1+len(keys)*2)
	args = append(args, sourceField)
	for _, k := range keys {
		path := "'$." + strings.ReplaceAll(k, "->", ".") + "'"
		val := fmt.Sprintf("%v", jsql.Quoted(source[k]))
		args = append(args, path, val)
	}
	return fmt.Sprintf("json_set(%s)", strings.Join(args, ", "))
}

/**
* sqColsVals: Separates data into sorted parallel (column names, quoted value strings) slices
* and a source et.Json for ATTRIB fields destined for the _source TEXT (JSON) column.
* When excludePKs is true, primary key columns are omitted (for SET clauses).
* @param model *jsql.Model, data et.Json, excludePKs bool
* @return []string, []string, et.Json
**/
func sqColsVals(model *jsql.Model, data et.Json, excludePKs bool) (cols, vals []string, source et.Json) {
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
* sqAttribReturn: Builds a json_extract expression for RETURNING clauses (no table alias).
* @param sourceField string, field string, tp jsql.TypeData
* @return string
**/
func sqAttribReturn(sourceField, field string, tp jsql.TypeData) string {
	extract := fmt.Sprintf("json_extract(%s, '$.%s')", sourceField, field)
	if cast := sqAttribCast(tp); cast != "" {
		return fmt.Sprintf("CAST(%s AS %s) AS %s", extract, cast, field)
	}
	return fmt.Sprintf("%s AS %s", extract, field)
}

/**
* sqReturningClause: Builds the RETURNING clause, expanding ATTRIB columns from _source inline.
* Uses command.Returns when set; otherwise auto-builds from the model column list.
* Requires SQLite >= 3.35.0.
* @param command *jsql.Command
* @return string
**/
func sqReturningClause(command *jsql.Command) string {
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
			exprs = append(exprs, sqAttribReturn(model.SourceField, col.Name, col.TypeData))
		}
	}

	if len(exprs) == 0 {
		return "\nRETURNING *"
	}
	return "\nRETURNING " + strings.Join(exprs, ", ")
}

/**
* sqPKWhere: Builds a WHERE clause using primary key values from data.
* @param model *jsql.Model, data et.Json
* @return string
**/
func sqPKWhere(model *jsql.Model, data et.Json) string {
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
* sqInsertSQL: Generates INSERT INTO … (cols) VALUES (vals) RETURNING …
* @param command *jsql.Command
* @return string, error
**/
func sqInsertSQL(command *jsql.Command) (string, error) {
	table := sqFromRef(command.From)
	model := command.From.Model

	var cols, vals []string
	var source et.Json

	if model != nil {
		cols, vals, source = sqColsVals(model, command.New, false)
		if model.SourceField != "" && len(source) > 0 {
			cols = append(cols, model.SourceField)
			vals = append(vals, fmt.Sprintf("%v", jsql.Quoted(source)))
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
	sb.WriteString(sqReturningClause(command))
	sb.WriteString(";")
	return sb.String(), nil
}

/**
* sqUpdateSQL: Generates UPDATE … SET … WHERE … RETURNING …
* Excludes primary key columns from SET; WHERE uses PK values from command.New.
* ATTRIB fields are updated via json_set to patch individual keys in _source.
* @param command *jsql.Command
* @return string, error
**/
func sqUpdateSQL(command *jsql.Command) (string, error) {
	table := sqFromRef(command.From)
	model := command.From.Model

	var setCols []string

	if model != nil {
		cols, vals, source := sqColsVals(model, command.New, true)
		for i, col := range cols {
			setCols = append(setCols, fmt.Sprintf("%s = %s", col, vals[i]))
		}
		if model.SourceField != "" && len(source) > 0 {
			setCols = append(setCols, fmt.Sprintf("%s = %s", model.SourceField, sqJsonSet(model.SourceField, source)))
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
		whereSQL = sqPKWhere(model, command.New)
	}
	if whereSQL == "" && model != nil && len(command.Conditions) > 0 {
		whereSQL = sqCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL != "" {
		sb.WriteString("\nWHERE " + whereSQL)
	}

	sb.WriteString(sqReturningClause(command))
	sb.WriteString(";")
	return sb.String(), nil
}

/**
* sqDeleteSQL: Generates DELETE FROM … WHERE … RETURNING …
* WHERE uses primary key values from command.Old (the fetched row).
* @param command *jsql.Command
* @return string, error
**/
func sqDeleteSQL(command *jsql.Command) (string, error) {
	table := sqFromRef(command.From)
	model := command.From.Model

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("DELETE FROM %s", table))

	var whereSQL string
	if model != nil && len(model.PrimaryKeys) > 0 && len(command.Old) > 0 {
		whereSQL = sqPKWhere(model, command.Old)
	}
	if whereSQL == "" && model != nil && len(command.Conditions) > 0 {
		whereSQL = sqCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL != "" {
		sb.WriteString("\nWHERE " + whereSQL)
	}

	sb.WriteString(sqReturningClause(command))
	sb.WriteString(";")
	return sb.String(), nil
}

/**
* Command: Generates the SQL DML string (INSERT, UPDATE, DELETE, BULK) for the given Command (SQLite dialect).
* @param command *jsql.Command
* @return string, error
**/
func (s *Sqlite) Command(command *jsql.Command) (string, error) {
	switch command.Type {
	case jsql.INSERT, jsql.BULK:
		return sqInsertSQL(command)
	case jsql.UPDATE:
		return sqUpdateSQL(command)
	case jsql.DELETE:
		return sqDeleteSQL(command)
	default:
		return "", fmt.Errorf("unsupported command type: %s", command.Type)
	}
}
