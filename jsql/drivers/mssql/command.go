package mssql

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* msJsonModify: Builds a chained JSON_MODIFY expression to patch individual ATTRIB keys.
* SQL Server's JSON_MODIFY sets one path at a time, so calls are chained like jsonb_set.
* @param sourceField string, source et.Json
* @return string
**/
func msJsonModify(sourceField string, source et.Json) string {
	keys := make([]string, 0, len(source))
	for k := range source {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	expr := sourceField
	for _, k := range keys {
		path := "'$." + strings.ReplaceAll(k, "->", ".") + "'"
		val := fmt.Sprintf("%v", jsql.Quoted(source[k]))
		expr = fmt.Sprintf("JSON_MODIFY(%s, %s, %s)", expr, path, val)
	}
	return expr
}

/**
* msColsVals: Separates data into sorted parallel (column names, quoted value strings) slices
* and a source et.Json for ATTRIB fields destined for the _source NVARCHAR(MAX) column.
* When excludePKs is true, primary key columns are omitted (for SET clauses).
* @param model *jsql.Model, data et.Json, excludePKs bool
* @return []string, []string, et.Json
**/
func msColsVals(model *jsql.Model, data et.Json, excludePKs bool) (cols, vals []string, source et.Json) {
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
* msAttribOutput: Builds a JSON_VALUE expression for the OUTPUT clause.
* @param prefix string ("INSERTED" or "DELETED"), sourceField string, field string, tp jsql.TypeData
* @return string
**/
func msAttribOutput(prefix, sourceField, field string, tp jsql.TypeData) string {
	extract := fmt.Sprintf("JSON_VALUE(%s.[%s], '$.%s')", prefix, sourceField, field)
	switch tp {
	case jsql.INT:
		return fmt.Sprintf("CAST(%s AS BIGINT) AS [%s]", extract, field)
	case jsql.FLOAT:
		return fmt.Sprintf("CAST(%s AS FLOAT) AS [%s]", extract, field)
	case jsql.BOOLEAN:
		return fmt.Sprintf("CAST(%s AS BIT) AS [%s]", extract, field)
	case jsql.DATETIME:
		return fmt.Sprintf("CAST(%s AS DATETIME2(7)) AS [%s]", extract, field)
	default:
		return fmt.Sprintf("%s AS [%s]", extract, field)
	}
}

/**
* msOutputClause: Builds the OUTPUT clause for INSERT/UPDATE/DELETE.
* Uses INSERTED.* prefix for INSERT and UPDATE, DELETED.* for DELETE.
* @param command *jsql.Command, prefix string
* @return string
**/
func msOutputClause(command *jsql.Command, prefix string) string {
	if len(command.Returns) > 0 {
		prefixed := make([]string, len(command.Returns))
		for i, r := range command.Returns {
			prefixed[i] = prefix + "." + r
		}
		return "\nOUTPUT " + strings.Join(prefixed, ", ")
	}

	model := command.From.Model
	if model == nil {
		return fmt.Sprintf("\nOUTPUT %s.*", prefix)
	}

	exprs := make([]string, 0, len(model.Columns))
	for _, col := range model.Columns {
		switch col.TypeColumn {
		case jsql.COLUMN:
			if col.Name == model.SourceField {
				continue
			}
			exprs = append(exprs, fmt.Sprintf("%s.[%s]", prefix, col.Name))
		case jsql.ATTRIB:
			if model.SourceField == "" {
				continue
			}
			exprs = append(exprs, msAttribOutput(prefix, model.SourceField, col.Name, col.TypeData))
		}
	}

	if len(exprs) == 0 {
		return fmt.Sprintf("\nOUTPUT %s.*", prefix)
	}
	return "\nOUTPUT " + strings.Join(exprs, ", ")
}

/**
* msPKWhere: Builds a WHERE clause using primary key values from data.
* @param model *jsql.Model, data et.Json
* @return string
**/
func msPKWhere(model *jsql.Model, data et.Json) string {
	conds := make([]string, 0, len(model.PrimaryKeys))
	for _, pk := range model.PrimaryKeys {
		val, ok := data[pk.Name]
		if !ok {
			continue
		}
		conds = append(conds, fmt.Sprintf("[%s] = %v", pk.Name, jsql.Quoted(val)))
	}
	return strings.Join(conds, " AND ")
}

/**
* msInsertSQL: Generates INSERT INTO … (cols) OUTPUT INSERTED.* VALUES (vals)
* The OUTPUT clause expands ATTRIB columns from _source via JSON_VALUE.
* @param command *jsql.Command
* @return string, error
**/
func msInsertSQL(command *jsql.Command) (string, error) {
	table := msFromRef(command.From)
	model := command.From.Model

	var cols, vals []string
	var source et.Json

	if model != nil {
		cols, vals, source = msColsVals(model, command.New, false)
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
		quotedCols[i] = fmt.Sprintf("[%s]", c)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO %s\n", table))
	sb.WriteString(fmt.Sprintf("  (%s)", strings.Join(quotedCols, ", ")))
	sb.WriteString(msOutputClause(command, "INSERTED"))
	sb.WriteString(fmt.Sprintf("\nVALUES\n  (%s);", strings.Join(vals, ", ")))
	return sb.String(), nil
}

/**
* msUpdateSQL: Generates UPDATE … SET … OUTPUT INSERTED.* WHERE …
* ATTRIB fields are updated via chained JSON_MODIFY to patch individual keys in _source.
* @param command *jsql.Command
* @return string, error
**/
func msUpdateSQL(command *jsql.Command) (string, error) {
	table := msFromRef(command.From)
	model := command.From.Model

	var setCols []string

	if model != nil {
		cols, vals, source := msColsVals(model, command.New, true)
		for i, col := range cols {
			setCols = append(setCols, fmt.Sprintf("[%s] = %s", col, vals[i]))
		}
		if model.SourceField != "" && len(source) > 0 {
			setCols = append(setCols, fmt.Sprintf("[%s] = %s", model.SourceField, msJsonModify(model.SourceField, source)))
		}
	} else {
		keys := make([]string, 0, len(command.New))
		for k := range command.New {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			setCols = append(setCols, fmt.Sprintf("[%s] = %v", k, jsql.Quoted(command.New[k])))
		}
	}

	if len(setCols) == 0 {
		return "", fmt.Errorf("no columns to update in %s", table)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("UPDATE %s\n", table))
	sb.WriteString("SET\n  " + strings.Join(setCols, ",\n  "))
	sb.WriteString(msOutputClause(command, "INSERTED"))

	var whereSQL string
	if model != nil && len(model.PrimaryKeys) > 0 {
		whereSQL = msPKWhere(model, command.New)
	}
	if whereSQL == "" && model != nil && len(command.Conditions) > 0 {
		whereSQL = msCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL != "" {
		sb.WriteString("\nWHERE " + whereSQL)
	}

	sb.WriteString(";")
	return sb.String(), nil
}

/**
* msDeleteSQL: Generates DELETE FROM … OUTPUT DELETED.* WHERE …
* WHERE uses primary key values from command.Old (the fetched row).
* @param command *jsql.Command
* @return string, error
**/
func msDeleteSQL(command *jsql.Command) (string, error) {
	table := msFromRef(command.From)
	model := command.From.Model

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("DELETE FROM %s", table))
	sb.WriteString(msOutputClause(command, "DELETED"))

	var whereSQL string
	if model != nil && len(model.PrimaryKeys) > 0 && len(command.Old) > 0 {
		whereSQL = msPKWhere(model, command.Old)
	}
	if whereSQL == "" && model != nil && len(command.Conditions) > 0 {
		whereSQL = msCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL != "" {
		sb.WriteString("\nWHERE " + whereSQL)
	}

	sb.WriteString(";")
	return sb.String(), nil
}

/**
* Command: Generates the T-SQL DML string (INSERT, UPDATE, DELETE, BULK) for the given Command.
* @param command *jsql.Command
* @return string, error
**/
func (s *MSSQL) Command(command *jsql.Command) (string, error) {
	switch command.Type {
	case jsql.INSERT, jsql.BULK:
		return msInsertSQL(command)
	case jsql.UPDATE:
		return msUpdateSQL(command)
	case jsql.DELETE:
		return msDeleteSQL(command)
	default:
		return "", fmt.Errorf("unsupported command type: %s", command.Type)
	}
}
