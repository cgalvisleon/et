package postgres

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
)

/**
* pgJsonbValue: Serializes a Go value as a PostgreSQL jsonb literal ('...'::jsonb),
* escaping single quotes so any string content is safe inside the literal.
* @param val any
* @return string
**/
func pgJsonbValue(val any) string {
	str, err := jsql.JsonString(val)
	if err != nil {
		logs.Errorf("pgJsonbValue, error marshalling value:%v, error:%v", val, err)
		return "'null'::jsonb"
	}
	return fmt.Sprintf("'%s'::jsonb", jsql.EscapeSQLString(str))
}

/**
* pgJsonbSetPath: Converts a '->' separated field path into a PostgreSQL jsonb path literal ('{"a","b"}').
* @param field string
* @return string
**/
func pgJsonbSetPath(field string) string {
	return pgTextArray(strings.Split(field, "->"))
}

/**
* pgJsonbSet: Builds the expression that merges the ATTRIB keys of source into sourceField
* without removing the other keys already stored there.
* Top-level keys are merged with ||; nested keys ("a->b") use jsonb_set, creating the missing
* parent objects first. A NULL sourceField is treated as '{}'.
* @param sourceField string, source et.Json
* @return string
**/
func pgJsonbSet(sourceField string, source et.Json) string {
	top := et.Json{}
	nested := make([]string, 0)
	for k, v := range source {
		if strings.Contains(k, "->") {
			nested = append(nested, k)
		} else {
			top[k] = v
		}
	}
	sort.Strings(nested)

	expr := fmt.Sprintf("COALESCE(%s, '{}'::jsonb)", sourceField)
	if len(top) > 0 {
		expr = fmt.Sprintf("%s || %s", expr, pgJsonbValue(top))
	}

	parents := make([]string, 0)
	for _, k := range nested {
		parts := strings.Split(k, "->")
		for i := 1; i < len(parts); i++ {
			parent := strings.Join(parts[:i], "->")
			if !slices.Contains(parents, parent) {
				parents = append(parents, parent)
			}
		}
	}
	sort.SliceStable(parents, func(i, j int) bool {
		return strings.Count(parents[i], "->") < strings.Count(parents[j], "->")
	})
	for _, parent := range parents {
		if _, ok := top[strings.Split(parent, "->")[0]]; ok {
			continue
		}
		path := pgJsonbSetPath(parent)
		expr = fmt.Sprintf("jsonb_set(%s, %s, COALESCE(%s #> %s, '{}'::jsonb), true)", expr, path, sourceField, path)
	}

	for _, k := range nested {
		expr = fmt.Sprintf("jsonb_set(%s, %s, %s, true)", expr, pgJsonbSetPath(k), pgJsonbValue(source[k]))
	}
	return expr
}

/**
* pgNestSource: Expands '->' keys of source into nested objects ({"a->b": 1} → {"a": {"b": 1}}),
* so an INSERT stores the same shape an UPDATE produces with jsonb_set.
* @param source et.Json
* @return et.Json
**/
func pgNestSource(source et.Json) et.Json {
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
* pgSortedKeys: The keys of data in order, so the generated SQL is always the same.
* @param data et.Json
* @return []string
**/
func pgSortedKeys(data et.Json) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

/**
* pgAttribValue: The SQL value of an attribute stored at the root: objects and lists as jsonb, the rest quoted.
* @param val any
* @return string
**/
func pgAttribValue(val any) string {
	switch val.(type) {
	case et.Json, map[string]any, []any, []et.Json:
		return pgJsonbValue(val)
	}
	return fmt.Sprintf("%v", jsql.Quoted(val))
}

/**
* pgSetPaths: A copy of the object val with each "a->b" key of paths set, creating the missing parents. A value
* that is not an object cannot hold nested keys and is returned as it is.
* @param val any, paths et.Json
* @return any
**/
func pgSetPaths(val any, paths et.Json) any {
	var obj et.Json
	switch v := val.(type) {
	case et.Json:
		obj = v.DeepClone()
	case map[string]any:
		obj = et.Json(v).DeepClone()
	default:
		return val
	}
	for _, k := range pgSortedKeys(paths) {
		parts := strings.Split(k, "->")
		node := obj
		for _, part := range parts[:len(parts)-1] {
			var child et.Json
			switch c := node[part].(type) {
			case et.Json:
				child = c
			case map[string]any:
				child = et.Json(c)
			default:
				child = et.Json{}
			}
			node[part] = child
			node = child
		}
		node[parts[len(parts)-1]] = paths[k]
	}
	return obj
}

/**
* pgRootSets:The SET items of an UPDATE for attributes stored at the root: "a" = value for a top-level key, and
* for nested keys ("a->b") the root field merged with jsonb_set, so the other keys of "a" stay. When the same data
* also sets the root whole (an update carries the merged record), the nested keys are applied over that value.
* @param source et.Json
* @return []string
**/
func pgRootSets(source et.Json) []string {
	top := et.Json{}
	nested := map[string]et.Json{}
	for k, v := range source {
		root, rest, ok := strings.Cut(k, "->")
		if !ok {
			top[k] = v
			continue
		}
		if nested[root] == nil {
			nested[root] = et.Json{}
		}
		nested[root][rest] = v
	}

	result := []string{}
	for _, k := range pgSortedKeys(top) {
		val := top[k]
		if paths, ok := nested[k]; ok {
			val = pgSetPaths(val, paths)
		}
		result = append(result, fmt.Sprintf("%s = %s", pgIdent(k), pgAttribValue(val)))
	}
	roots := make([]string, 0, len(nested))
	for root := range nested {
		if _, ok := top[root]; !ok {
			roots = append(roots, root)
		}
	}
	sort.Strings(roots)
	for _, root := range roots {
		result = append(result, fmt.Sprintf("%s = %s", pgIdent(root), pgJsonbSet(pgIdent(root), nested[root])))
	}
	return result
}

/**
* pgColsVals: Separates data into sorted parallel (column names, quoted value strings) slices
* and a source et.Json for ATTRIB fields destined for the _source JSONB column.
* When excludePKs is true, primary key columns are omitted from the column lists (for SET clauses).
* @param model *jsql.Model, data et.Json, excludePKs bool
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
* pgReturningClause: Builds the RETURNING clause with the same shape as a SELECT without fields:
* with a SourceField the row comes back as one JSON (sourceField || jsonb_build_object(columns) AS result),
* so attributes without a declared column are returned too; without it, the column list.
* Hidden columns and attributes are excluded. Uses command.Returns when set.
* @param command *jsql.Command
* @return string
**/
func pgReturningClause(command *jsql.Command) string {
	model := command.From.Model
	if model == nil {
		if len(command.Returns) > 0 {
			cols := make([]string, len(command.Returns))
			for i, name := range command.Returns {
				cols[i] = sanitizeIdent(name)
			}
			return "\nRETURNING " + strings.Join(cols, ", ")
		}
		return "\nRETURNING *"
	}

	if len(command.Returns) > 0 {
		return pgReturningFields(model, command.Returns)
	}

	pairs := make([]string, 0, len(model.Columns))
	cols := make([]string, 0, len(model.Columns))
	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN || col.Name == model.SourceField {
			continue
		}
		if slices.Contains(model.Hiddens, col.Name) {
			continue
		}
		pairs = append(pairs, fmt.Sprintf("%s, %s", pgQuoteKey(col.Name), col.Name))
		cols = append(cols, col.Name)
	}

	if model.SourceField != "" {
		source := pgSourceExpr(model.SourceField, model.Hiddens)
		if jsql.AttribsAtRoot(model) {
			source = pgRowExpr(model, pgTableRef(model.Table), model.Hiddens)
		}
		return fmt.Sprintf("\nRETURNING %s AS %s", pgMergeObject(source, pairs), jsql.RESULT)
	}

	if len(cols) == 0 {
		return "\nRETURNING *"
	}
	return "\nRETURNING " + strings.Join(cols, ", ")
}

/**
* pgReturningFields: Builds the RETURNING clause of Command.Return(fields...) resolving each field like a
* Select: columns, SourceField attributes, "field:as" aliases and nested paths. With a SourceField the row
* comes back as one JSON "result" with those fields; without it, as the listed columns.
* @param model *jsql.Model, fields []string
* @return string
**/
func pgReturningFields(model *jsql.Model, fields []string) string {
	// The query has no alias (its As is the table), so the expressions are unqualified as RETURNING needs.
	query := jsql.NewQuery(model, model.Table)
	exprs := make([]string, 0, len(fields))
	for _, field := range fields {
		if expr, ok := pgSelectExpr(query, field); ok {
			exprs = append(exprs, expr)
		}
	}
	if len(exprs) == 0 {
		return "\nRETURNING *"
	}
	if query.UseSourceField {
		return fmt.Sprintf("\nRETURNING %s AS %s", pgMergeObject("", exprs), jsql.RESULT)
	}
	return "\nRETURNING " + strings.Join(exprs, ", ")
}

/**
* pgPKWhere: Builds a WHERE clause using primary key values from data.
* Returns empty string when any primary key is missing, so the caller falls back to Conditions.
* @param model *jsql.Model, data et.Json
* @return string
**/
func pgPKWhere(model *jsql.Model, data et.Json) string {
	conds := make([]string, 0, len(model.PrimaryKeys))
	for _, pk := range model.PrimaryKeys {
		val, ok := data[pk.Name]
		if !ok {
			return ""
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
			if jsql.AttribsAtRoot(model) {
				// Each attribute is a field of the document's root
				nested := pgNestSource(source)
				for _, key := range pgSortedKeys(nested) {
					cols = append(cols, pgIdent(key))
					vals = append(vals, pgAttribValue(nested[key]))
				}
			} else {
				cols = append(cols, model.SourceField)
				vals = append(vals, pgJsonbValue(pgNestSource(source)))
			}
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
			cols[i] = sanitizeIdent(k)
			vals[i] = fmt.Sprintf("%v", jsql.Quoted(command.New[k]))
		}
	}

	if len(cols) == 0 {
		return "", fmt.Errorf(jsql.MSG_NO_COLUMNS_INSERT, table)
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
* Excludes primary key columns from SET; WHERE uses the PK values of the fetched row (command.Old),
* falling back to command.New and then to the command Conditions.
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
			if jsql.AttribsAtRoot(model) {
				setCols = append(setCols, pgRootSets(source)...)
			} else {
				setCols = append(setCols, fmt.Sprintf("%s = %s", model.SourceField, pgJsonbSet(model.SourceField, source)))
			}
		}
	} else {
		keys := make([]string, 0, len(command.New))
		for k := range command.New {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			setCols = append(setCols, fmt.Sprintf("%s = %v", sanitizeIdent(k), jsql.Quoted(command.New[k])))
		}
	}

	if len(setCols) == 0 {
		return "", fmt.Errorf(jsql.MSG_NO_COLUMNS_UPDATE, table)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("UPDATE %s\n", table))
	sb.WriteString("SET\n  " + strings.Join(setCols, ",\n  "))

	var whereSQL string
	if model != nil && len(model.PrimaryKeys) > 0 {
		whereSQL = pgPKWhere(model, command.Old)
		if whereSQL == "" {
			whereSQL = pgPKWhere(model, command.New)
		}
	}
	if whereSQL == "" && model != nil && len(command.Conditions) > 0 {
		whereSQL = pgCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL == "" {
		return "", fmt.Errorf(jsql.MSG_REFUSE_WITHOUT_WHERE, "UPDATE", table)
	}
	sb.WriteString("\nWHERE " + whereSQL)

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
	if whereSQL == "" && model != nil && len(command.Conditions) > 0 {
		whereSQL = pgCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL == "" {
		return "", fmt.Errorf(jsql.MSG_REFUSE_WITHOUT_WHERE, "DELETE FROM", table)
	}
	sb.WriteString("\nWHERE " + whereSQL)

	sb.WriteString(pgReturningClause(command))
	sb.WriteString(";")
	return sb.String(), nil
}

/**
* Command: Generates the SQL DML string (INSERT, UPDATE, DELETE, BULK) for the given Command.
* @param command *jsql.Command
* @return string, error
**/
func (s *Postgres) Command(command *jsql.Command, timeout ...time.Duration) (string, error) {
	switch command.Type {
	case jsql.INSERT, jsql.BULK:
		return pgInsertSQL(command)
	case jsql.UPDATE:
		return pgUpdateSQL(command)
	case jsql.DELETE:
		return pgDeleteSQL(command)
	default:
		return "", fmt.Errorf(jsql.MSG_UNSUPPORTED_COMMAND, command.Type)
	}
}
