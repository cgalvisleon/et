package mysql

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* myNestSource: Expands '->' keys of source into nested objects ({"a->b": 1} → {"a": {"b": 1}}).
* @param source et.Json
* @return et.Json
**/
func myNestSource(source et.Json) et.Json {
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
* myJsonSet: Builds the expression that writes the ATTRIB keys of source into sourceField without removing
* the other keys: JSON_SET for each key, creating first the parent objects of nested keys ("a->b"), which
* JSON_SET does not create. JSON_SET keeps null values; a NULL source counts as {}.
* @param sourceField string, source et.Json
* @return string
**/
func myJsonSet(sourceField string, source et.Json) string {
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

	pairs := make([]string, 0, len(parents)+len(keys))
	for _, parent := range parents {
		path := myJsonPath(strings.Split(parent, "->"))
		pairs = append(pairs, fmt.Sprintf("%s, COALESCE(JSON_EXTRACT(%s, %s), JSON_OBJECT())", path, sourceField, path))
	}
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s, %s", myJsonPath(strings.Split(k, "->")), myJsonValue(source[k])))
	}
	return mySetObject(fmt.Sprintf("COALESCE(%s, JSON_OBJECT())", sourceField), pairs)
}

/**
* myColsVals: Separates data into sorted (quoted column, quoted value) slices, quoting each value for its
* column type, and a source et.Json for ATTRIB fields. excludePKs omits the primary keys (SET clauses).
* @param model *jsql.Model, data et.Json, excludePKs bool
* @return []string, []string, et.Json
**/
func myColsVals(model *jsql.Model, data et.Json, excludePKs bool) (cols, vals []string, source et.Json) {
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
			if excludePKs && slices.ContainsFunc(model.PrimaryKeys, func(pk *jsql.Index) bool { return pk.Name == key }) {
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
		cols[i] = myIdent(name)
		vals[i] = QuotedAs(columns[name].TypeData, data[name])
	}
	return
}

/**
* myRowResult: Builds the SELECT expression that returns an affected row as one JSON "result", with the
* same shape as a SELECT without fields, or only the fields of command.Returns (resolved like a Select).
* @param command *jsql.Command
* @return string
**/
func myRowResult(command *jsql.Command) string {
	model := command.From.Model
	if len(command.Returns) > 0 {
		query := jsql.NewQuery(model, model.Table)
		pairs := make([]string, 0, len(command.Returns))
		for _, field := range command.Returns {
			if expr, ok := mySelectExpr(query, field); ok {
				pairs = append(pairs, expr)
			}
		}
		return fmt.Sprintf("JSON_OBJECT(%s) AS %s", strings.Join(pairs, ", "), myIdent(jsql.RESULT))
	}
	return fmt.Sprintf("%s AS %s", myRowObject(command.From, nil), myIdent(jsql.RESULT))
}

/**
* myPKWhere: Builds a WHERE from the primary key values in data ("" when any is missing).
* @param model *jsql.Model, data et.Json
* @return string
**/
func myPKWhere(model *jsql.Model, data et.Json) string {
	conds := make([]string, 0, len(model.PrimaryKeys))
	for _, pk := range model.PrimaryKeys {
		val, ok := data[pk.Name]
		if !ok {
			return ""
		}
		conds = append(conds, fmt.Sprintf("%s = %s", myIdent(pk.Name), QuotedAs(columnType(model, pk.Name), val)))
	}
	return strings.Join(conds, " AND ")
}

/**
* myWhere: Returns the WHERE for a command: the primary key of the first row that has it, otherwise the
* command Conditions.
* @param command *jsql.Command, rows ...et.Json
* @return string, error
**/
func myWhere(command *jsql.Command, rows ...et.Json) (string, error) {
	model := command.From.Model
	if len(model.PrimaryKeys) > 0 {
		for _, row := range rows {
			if where := myPKWhere(model, row); where != "" {
				return where, nil
			}
		}
	}
	if len(command.Conditions) > 0 {
		if where := myConds(model.GetField, command.Conditions, "", false); where != "" {
			return where, nil
		}
	}
	return "", fmt.Errorf("refusing to %s %s without a WHERE clause (missing primary key value and no Conditions set)", strings.ToUpper(string(command.Type)), myFromRef(command.From))
}

/**
* myInsertSQL: INSERT followed by the SELECT of the inserted row (by its primary key).
* @param command *jsql.Command
* @return string, error
**/
func myInsertSQL(command *jsql.Command) (string, error) {
	table := myFromRef(command.From)
	model := command.From.Model
	cols, vals, source := myColsVals(model, command.New, false)
	if model.SourceField != "" && len(source) > 0 {
		cols = append(cols, myIdent(model.SourceField))
		vals = append(vals, myQuoteJson(myNestSource(source)))
	}
	if len(cols) == 0 {
		return "", fmt.Errorf("no columns to insert into %s", table)
	}
	sql := fmt.Sprintf("INSERT INTO %s\n  (%s)\n  VALUES (%s);", table, strings.Join(cols, ", "), strings.Join(vals, ", "))
	if where := myPKWhere(model, command.New); where != "" {
		sql += fmt.Sprintf("\nSELECT %s FROM %s WHERE %s;", myRowResult(command), table, where)
	}
	return sql, nil
}

/**
* myUpdateSQL: UPDATE followed by the SELECT of the updated row. Attributes are merged into the SourceField
* with JSON_SET, keeping the other keys.
* @param command *jsql.Command
* @return string, error
**/
func myUpdateSQL(command *jsql.Command) (string, error) {
	table := myFromRef(command.From)
	model := command.From.Model
	cols, vals, source := myColsVals(model, command.New, true)
	sets := make([]string, 0, len(cols)+1)
	for i, col := range cols {
		sets = append(sets, fmt.Sprintf("%s = %s", col, vals[i]))
	}
	if model.SourceField != "" && len(source) > 0 {
		src := myIdent(model.SourceField)
		sets = append(sets, fmt.Sprintf("%s = %s", src, myJsonSet(src, source)))
	}
	if len(sets) == 0 {
		return "", fmt.Errorf("no columns to update in %s", table)
	}
	where, err := myWhere(command, command.Old, command.New)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("UPDATE %s\n  SET %s\n  WHERE %s;\nSELECT %s FROM %s WHERE %s;",
		table, strings.Join(sets, ",\n    "), where, myRowResult(command), table, where), nil
}

/**
* myDeleteSQL: SELECT of the rows that are deleted, followed by the DELETE.
* @param command *jsql.Command
* @return string, error
**/
func myDeleteSQL(command *jsql.Command) (string, error) {
	table := myFromRef(command.From)
	where, err := myWhere(command, command.Old)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("SELECT %s FROM %s WHERE %s;\nDELETE FROM %s WHERE %s;",
		myRowResult(command), table, where, table, where), nil
}

/**
* Command: Generates the batch of a command (INSERT, BULK, UPDATE, DELETE); its SELECT returns the
* affected rows, since MySQL has no RETURNING.
* @param command *jsql.Command
* @return string, error
**/
func (s *Mysql) Command(command *jsql.Command) (string, error) {
	if command.From == nil || command.From.Model == nil {
		return "", fmt.Errorf("command without model")
	}
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
