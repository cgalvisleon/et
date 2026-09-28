package mssql

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* msNestSource: Expands '->' keys of source into nested objects ({"a->b": 1} → {"a": {"b": 1}}).
* @param source et.Json
* @return et.Json
**/
func msNestSource(source et.Json) et.Json {
	result := et.Json{}
	keys := make([]string, 0, len(source))
	for k := range source {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts := strings.Split(k, "->")
		node := result
		for _, part := range parts[:len(parts)-1] {
			child, ok := node[part].(et.Json)
			if !ok {
				child = et.Json{}
				node[part] = child
			}
			node = child
		}
		node[parts[len(parts)-1]] = source[k]
	}
	return result
}

/**
* msJsonModify: Builds the expression that writes the ATTRIB keys of source into sourceField without
* removing the other keys: JSON_MODIFY for each key, creating first the missing parent objects of nested
* keys ("a->b"). A NULL source counts as {}; a null value removes its key (JSON_MODIFY in lax mode).
* @param sourceField string, source et.Json
* @return string
**/
func msJsonModify(sourceField string, source et.Json) string {
	keys := make([]string, 0, len(source))
	for k := range source {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parents := make([]string, 0)
	for _, k := range keys {
		parts := strings.Split(k, "->")
		for i := 1; i < len(parts); i++ {
			parent := strings.Join(parts[:i], "->")
			if _, top := source[parent]; !top && !slices.Contains(parents, parent) {
				parents = append(parents, parent)
			}
		}
	}
	sort.SliceStable(parents, func(i, j int) bool {
		return strings.Count(parents[i], "->") < strings.Count(parents[j], "->")
	})

	expr := fmt.Sprintf("COALESCE(%s, N'{}')", sourceField)
	for _, parent := range parents {
		path := msJsonPath(strings.Split(parent, "->"))
		expr = fmt.Sprintf("JSON_MODIFY(%s, %s, JSON_QUERY(COALESCE(JSON_QUERY(%s, %s), N'{}')))", expr, path, sourceField, path)
	}
	for _, k := range keys {
		expr = fmt.Sprintf("JSON_MODIFY(%s, %s, %s)", expr, msJsonPath(strings.Split(k, "->")), msModifyValue(source[k]))
	}
	return expr
}

/**
* msColsVals: Separates data into sorted (quoted column, quoted value) slices, quoting each value for its
* column type, and a source et.Json for ATTRIB fields. excludePKs omits the primary keys (SET clauses).
* @param model *jsql.Model, data et.Json, excludePKs bool
* @return []string, []string, et.Json
**/
func msColsVals(model *jsql.Model, data et.Json, excludePKs bool) (cols, vals []string, source et.Json) {
	source = et.Json{}
	columns := make(map[string]*jsql.Column)
	for key := range data {
		if key == model.SourceField {
			continue
		}
		col, ok := model.GetColumn(key)
		if !ok {
			continue
		}
		switch col.TypeColumn {
		case jsql.COLUMN:
			if excludePKs && isPrimaryKey(model, key) {
				continue
			}
			columns[key] = col
		case jsql.ATTRIB:
			source[key] = data[key]
		}
	}
	names := make([]string, 0, len(columns))
	for k := range columns {
		names = append(names, k)
	}
	sort.Strings(names)
	cols = make([]string, len(names))
	vals = make([]string, len(names))
	for i, name := range names {
		cols[i] = msIdent(name)
		vals[i] = QuotedAs(columns[name].TypeData, data[name])
	}
	return
}

/**
* msRowResult: Builds the SELECT expression that returns an affected row as JSON text in "result", with the
* same shape as a SELECT without fields, or only the fields of command.Returns (resolved like a Select).
* @param command *jsql.Command
* @return string
**/
func msRowResult(command *jsql.Command) string {
	model := command.From.Model
	if len(command.Returns) > 0 {
		query := jsql.NewQuery(model, model.Table)
		pairs := make([]string, 0, len(command.Returns))
		for _, field := range command.Returns {
			if expr, ok := msSelectExpr(query, field); ok {
				pairs = append(pairs, expr)
			}
		}
		return fmt.Sprintf("%s AS %s", msObject(pairs), msIdent(jsql.RESULT))
	}
	return fmt.Sprintf("%s AS %s", msRowObject(command.From, nil), msIdent(jsql.RESULT))
}

/**
* msPKWhere: Builds a WHERE from the primary key values in data ("" when any is missing).
* @param model *jsql.Model, data et.Json
* @return string
**/
func msPKWhere(model *jsql.Model, data et.Json) string {
	conds := make([]string, 0, len(model.PrimaryKeys))
	for _, pk := range model.PrimaryKeys {
		val, ok := data[pk.Name]
		if !ok {
			return ""
		}
		conds = append(conds, fmt.Sprintf("%s = %s", msIdent(pk.Name), QuotedAs(columnType(model, pk.Name), val)))
	}
	return strings.Join(conds, " AND ")
}

/**
* msWhere: Returns the WHERE for a command: the primary key of the first row that has it, otherwise the
* command Conditions.
* @param command *jsql.Command, rows ...et.Json
* @return string, error
**/
func msWhere(command *jsql.Command, rows ...et.Json) (string, error) {
	model := command.From.Model
	if len(model.PrimaryKeys) > 0 {
		for _, row := range rows {
			if where := msPKWhere(model, row); where != "" {
				return where, nil
			}
		}
	}
	if len(command.Conditions) > 0 {
		if where := msConds(model.GetField, command.Conditions, "", false); where != "" {
			return where, nil
		}
	}
	return "", fmt.Errorf("refusing to %s %s without a WHERE clause (missing primary key value and no Conditions set)", strings.ToUpper(string(command.Type)), msFromRef(command.From))
}

/**
* msInsertSQL: INSERT followed by the SELECT of the inserted row (by its primary key).
* @param command *jsql.Command
* @return string, error
**/
func msInsertSQL(command *jsql.Command) (string, error) {
	table := msFromRef(command.From)
	model := command.From.Model
	cols, vals, source := msColsVals(model, command.New, false)
	if model.SourceField != "" && len(source) > 0 {
		cols = append(cols, msIdent(model.SourceField))
		vals = append(vals, msQuoteJson(msNestSource(source)))
	}
	if len(cols) == 0 {
		return "", fmt.Errorf("no columns to insert into %s", table)
	}
	sql := fmt.Sprintf("SET NOCOUNT ON;\nINSERT INTO %s\n  (%s)\n  VALUES (%s);", table, strings.Join(cols, ", "), strings.Join(vals, ", "))
	if where := msPKWhere(model, command.New); where != "" {
		sql += fmt.Sprintf("\nSELECT %s FROM %s WHERE %s;", msRowResult(command), table, where)
	}
	return sql, nil
}

/**
* msUpdateSQL: UPDATE followed by the SELECT of the updated row; attributes are merged into the SourceField
* with JSON_MODIFY.
* @param command *jsql.Command
* @return string, error
**/
func msUpdateSQL(command *jsql.Command) (string, error) {
	table := msFromRef(command.From)
	model := command.From.Model
	cols, vals, source := msColsVals(model, command.New, true)
	sets := make([]string, 0, len(cols)+1)
	for i, col := range cols {
		sets = append(sets, fmt.Sprintf("%s = %s", col, vals[i]))
	}
	if model.SourceField != "" && len(source) > 0 {
		src := msIdent(model.SourceField)
		sets = append(sets, fmt.Sprintf("%s = %s", src, msJsonModify(src, source)))
	}
	if len(sets) == 0 {
		return "", fmt.Errorf("no columns to update in %s", table)
	}
	where, err := msWhere(command, command.Old, command.New)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("SET NOCOUNT ON;\nUPDATE %s\n  SET %s\n  WHERE %s;\nSELECT %s FROM %s WHERE %s;",
		table, strings.Join(sets, ",\n    "), where, msRowResult(command), table, where), nil
}

/**
* msDeleteSQL: SELECT of the rows that are deleted, followed by the DELETE.
* @param command *jsql.Command
* @return string, error
**/
func msDeleteSQL(command *jsql.Command) (string, error) {
	table := msFromRef(command.From)
	where, err := msWhere(command, command.Old)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("SET NOCOUNT ON;\nSELECT %s FROM %s WHERE %s;\nDELETE FROM %s WHERE %s;",
		msRowResult(command), table, where, table, where), nil
}

/**
* Command: Generates the batch of a command (INSERT, BULK, UPDATE, DELETE); its SELECT returns the
* affected rows as JSON text.
* @param command *jsql.Command
* @return string, error
**/
func (s *Mssql) Command(command *jsql.Command) (string, error) {
	if command.From == nil || command.From.Model == nil {
		return "", fmt.Errorf("command without model")
	}
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
