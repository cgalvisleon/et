package oracle

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* orJsonMergePatch: Builds a JSON_MERGEPATCH expression that patches individual ATTRIB keys
* into sourceField without overwriting the rest of the CLOB JSON document.
* Uses RFC 7396 top-level merge — sufficient for single-level ATTRIB keys.
* Single quotes inside the JSON literal are doubled for Oracle string safety.
* @param sourceField string, source et.Json
* @return string
**/
func orJsonMergePatch(sourceField string, source et.Json) string {
	bt, err := json.Marshal(source)
	if err != nil {
		return fmt.Sprintf("\"%s\"", sourceField)
	}
	jsonStr := strings.ReplaceAll(string(bt), "'", "''")
	return fmt.Sprintf("JSON_MERGEPATCH(\"%s\", '%s' RETURNING CLOB)", sourceField, jsonStr)
}

/**
* orColsVals: Separates data into sorted parallel (column names, quoted value strings) slices
* and a source et.Json for ATTRIB fields destined for the _source CLOB column.
* When excludePKs is true, primary key columns are omitted from the column lists (for SET clauses).
* @param model *jsql.Model, data et.Json, excludePKs bool
* @return []string, []string, et.Json
**/
func orColsVals(model *jsql.Model, data et.Json, excludePKs bool) (cols, vals []string, source et.Json) {
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
* orPKWhere: Builds a WHERE clause using primary key values from data.
* @param model *jsql.Model, data et.Json
* @return string
**/
func orPKWhere(model *jsql.Model, data et.Json) string {
	conds := make([]string, 0, len(model.PrimaryKeys))
	for _, pk := range model.PrimaryKeys {
		val, ok := data[pk.Name]
		if !ok {
			continue
		}
		conds = append(conds, fmt.Sprintf("\"%s\" = %v", pk.Name, jsql.Quoted(val)))
	}
	return strings.Join(conds, " AND ")
}

/**
* orInsertSQL: Generates INSERT INTO … (cols) VALUES (vals).
* Oracle does not support RETURNING with database/sql bind parameters, so no RETURNING clause
* is emitted; the execution layer uses command.New/command.Old for the result instead.
* @param command *jsql.Command
* @return string, error
**/
func orInsertSQL(command *jsql.Command) (string, error) {
	table := orFromRef(command.From)
	model := command.From.Model

	var cols, vals []string
	var source et.Json

	if model != nil {
		cols, vals, source = orColsVals(model, command.New, false)
		if model.SourceField != "" && len(source) > 0 {
			bt, err := json.Marshal(source)
			if err != nil {
				return "", err
			}
			jsonStr := strings.ReplaceAll(string(bt), "'", "''")
			cols = append(cols, model.SourceField)
			vals = append(vals, fmt.Sprintf("'%s'", jsonStr))
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
		quotedCols[i] = fmt.Sprintf("\"%s\"", c)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO %s\n", table))
	sb.WriteString(fmt.Sprintf("  (%s)\n", strings.Join(quotedCols, ", ")))
	sb.WriteString(fmt.Sprintf("VALUES\n  (%s)", strings.Join(vals, ", ")))
	return sb.String(), nil
}

/**
* orUpdateSQL: Generates UPDATE … SET … WHERE …
* ATTRIB fields are merged via JSON_MERGEPATCH; primary key columns are excluded from SET.
* @param command *jsql.Command
* @return string, error
**/
func orUpdateSQL(command *jsql.Command) (string, error) {
	table := orFromRef(command.From)
	model := command.From.Model

	var setCols []string

	if model != nil {
		cols, vals, source := orColsVals(model, command.New, true)
		for i, col := range cols {
			setCols = append(setCols, fmt.Sprintf("\"%s\" = %s", col, vals[i]))
		}
		if model.SourceField != "" && len(source) > 0 {
			setCols = append(setCols, fmt.Sprintf("\"%s\" = %s", model.SourceField, orJsonMergePatch(model.SourceField, source)))
		}
	} else {
		keys := make([]string, 0, len(command.New))
		for k := range command.New {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			setCols = append(setCols, fmt.Sprintf("\"%s\" = %v", k, jsql.Quoted(command.New[k])))
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
		whereSQL = orPKWhere(model, command.New)
	}
	if whereSQL == "" && model != nil && len(command.Conditions) > 0 {
		whereSQL = orCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL != "" {
		sb.WriteString("\nWHERE " + whereSQL)
	}

	return sb.String(), nil
}

/**
* orDeleteSQL: Generates DELETE FROM … WHERE …
* WHERE uses primary key values from command.Old (the fetched row).
* @param command *jsql.Command
* @return string, error
**/
func orDeleteSQL(command *jsql.Command) (string, error) {
	table := orFromRef(command.From)
	model := command.From.Model

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("DELETE FROM %s", table))

	var whereSQL string
	if model != nil && len(model.PrimaryKeys) > 0 && len(command.Old) > 0 {
		whereSQL = orPKWhere(model, command.Old)
	}
	if whereSQL == "" && model != nil && len(command.Conditions) > 0 {
		whereSQL = orCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL != "" {
		sb.WriteString("\nWHERE " + whereSQL)
	}

	return sb.String(), nil
}

/**
* Command: Generates the Oracle SQL DML string (INSERT, UPDATE, DELETE, BULK) for the given Command.
* @param command *jsql.Command
* @return string, error
**/
func (s *Oracle) Command(command *jsql.Command) (string, error) {
	switch command.Type {
	case jsql.INSERT, jsql.BULK:
		return orInsertSQL(command)
	case jsql.UPDATE:
		return orUpdateSQL(command)
	case jsql.DELETE:
		return orDeleteSQL(command)
	default:
		return "", fmt.Errorf("unsupported command type: %s", command.Type)
	}
}
