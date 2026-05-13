package mysql

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* myJsonSet: Builds a JSON_SET expression to patch individual ATTRIB keys into sourceField.
* MySQL's JSON_SET accepts multiple path-value pairs in a single call and creates missing keys.
* @param sourceField string, source et.Json
* @return string
**/
func myJsonSet(sourceField string, source et.Json) string {
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
	return fmt.Sprintf("JSON_SET(%s)", strings.Join(args, ", "))
}

/**
* myColsVals: Separates data into sorted parallel (column names, quoted value strings) slices
* and a source et.Json for ATTRIB fields destined for the _source JSON column.
* When excludePKs is true, primary key columns are omitted (for SET clauses).
* @param model *jsql.Model, data et.Json, excludePKs bool
* @return []string, []string, et.Json
**/
func myColsVals(model *jsql.Model, data et.Json, excludePKs bool) (cols, vals []string, source et.Json) {
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
* myPKWhere: Builds a WHERE clause using primary key values from data.
* @param model *jsql.Model, data et.Json
* @return string
**/
func myPKWhere(model *jsql.Model, data et.Json) string {
	conds := make([]string, 0, len(model.PrimaryKeys))
	for _, pk := range model.PrimaryKeys {
		val, ok := data[pk.Name]
		if !ok {
			continue
		}
		conds = append(conds, fmt.Sprintf("`%s` = %v", pk.Name, jsql.Quoted(val)))
	}
	return strings.Join(conds, " AND ")
}

/**
* myInsertSQL: Generates INSERT INTO … (cols) VALUES (vals)
* MySQL does not support RETURNING; results are provided by the execution layer from s.New.
* @param command *jsql.Command
* @return string, error
**/
func myInsertSQL(command *jsql.Command) (string, error) {
	table := myFromRef(command.From)
	model := command.From.Model

	var cols, vals []string
	var source et.Json

	if model != nil {
		cols, vals, source = myColsVals(model, command.New, false)
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

	quotedCols := make([]string, len(cols))
	for i, c := range cols {
		quotedCols[i] = fmt.Sprintf("`%s`", c)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO `%s`\n", table))
	sb.WriteString(fmt.Sprintf("  (%s)\n", strings.Join(quotedCols, ", ")))
	sb.WriteString(fmt.Sprintf("VALUES\n  (%s);", strings.Join(vals, ", ")))
	return sb.String(), nil
}

/**
* myUpdateSQL: Generates UPDATE … SET … WHERE …
* ATTRIB fields are updated via JSON_SET to patch individual keys in _source.
* MySQL does not support RETURNING; results come from the execution layer's s.New.
* @param command *jsql.Command
* @return string, error
**/
func myUpdateSQL(command *jsql.Command) (string, error) {
	table := myFromRef(command.From)
	model := command.From.Model

	var setCols []string

	if model != nil {
		cols, vals, source := myColsVals(model, command.New, true)
		for i, col := range cols {
			setCols = append(setCols, fmt.Sprintf("`%s` = %s", col, vals[i]))
		}
		if model.SourceField != "" && len(source) > 0 {
			setCols = append(setCols, fmt.Sprintf("`%s` = %s", model.SourceField, myJsonSet(model.SourceField, source)))
		}
	} else {
		keys := make([]string, 0, len(command.New))
		for k := range command.New {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			setCols = append(setCols, fmt.Sprintf("`%s` = %v", k, jsql.Quoted(command.New[k])))
		}
	}

	if len(setCols) == 0 {
		return "", fmt.Errorf("no columns to update in %s", table)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("UPDATE `%s`\n", table))
	sb.WriteString("SET\n  " + strings.Join(setCols, ",\n  "))

	var whereSQL string
	if model != nil && len(model.PrimaryKeys) > 0 {
		whereSQL = myPKWhere(model, command.New)
	}
	if whereSQL == "" && model != nil && len(command.Conditions) > 0 {
		whereSQL = myCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL != "" {
		sb.WriteString("\nWHERE " + whereSQL)
	}

	sb.WriteString(";")
	return sb.String(), nil
}

/**
* myDeleteSQL: Generates DELETE FROM … WHERE …
* WHERE uses primary key values from command.Old (the fetched row).
* MySQL does not support RETURNING; results come from the execution layer's s.Old.
* @param command *jsql.Command
* @return string, error
**/
func myDeleteSQL(command *jsql.Command) (string, error) {
	table := myFromRef(command.From)
	model := command.From.Model

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("DELETE FROM `%s`", table))

	var whereSQL string
	if model != nil && len(model.PrimaryKeys) > 0 && len(command.Old) > 0 {
		whereSQL = myPKWhere(model, command.Old)
	}
	if whereSQL == "" && model != nil && len(command.Conditions) > 0 {
		whereSQL = myCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL != "" {
		sb.WriteString("\nWHERE " + whereSQL)
	}

	sb.WriteString(";")
	return sb.String(), nil
}

/**
* Command: Generates the SQL DML string (INSERT, UPDATE, DELETE, BULK) for the given Command (MySQL dialect).
* @param command *jsql.Command
* @return string, error
**/
func (s *MySQL) Command(command *jsql.Command) (string, error) {
	switch command.Type {
	case jsql.INSERT, jsql.BULK:
		return myInsertSQL(command)
	case jsql.UPDATE:
		return myUpdateSQL(command)
	case jsql.DELETE:
		return myDeleteSQL(command)
	default:
		return "", fmt.Errorf("unsupported command type: %s", command.Type)
	}
}
