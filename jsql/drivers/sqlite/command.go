package sqlite

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* sqliteJsonValue: Serializes a Go value as a SQLite JSON text literal, escaping single quotes.
* @param val any
* @return string
**/
func sqliteJsonValue(val any) string {
	str, err := jsql.JsonString(val)
	if err != nil {
		return "'null'"
	}
	return fmt.Sprintf("'%s'", jsql.EscapeSQLString(str))
}

/**
* sqliteJsonSet: Builds the expression that writes the ATTRIB keys of source into sourceField
* without removing the other keys. Each key ("a" or "a->b") becomes a json_set path, which
* creates missing parents and keeps null values. A NULL sourceField is treated as '{}'.
* @param sourceField string, source et.Json
* @return string
**/
func sqliteJsonSet(sourceField string, source et.Json) string {
	keys := make([]string, 0, len(source))
	for k := range source {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = fmt.Sprintf("%s, json(%s)", sqlitePath(strings.Split(k, "->")), sqliteJsonValue(source[k]))
	}
	return sqliteSetObject(fmt.Sprintf("COALESCE(%s, '{}')", sourceField), pairs)
}

/**
* sqliteNestSource: Expands '->' keys of source into nested objects ({"a->b": 1} → {"a": {"b": 1}}),
* so an INSERT stores the same shape an UPDATE produces with json_set.
* @param source et.Json
* @return et.Json
**/
func sqliteNestSource(source et.Json) et.Json {
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
* sqliteColsVals: Separates data into sorted parallel (column names, quoted value
* strings) slices and a source et.Json for ATTRIB fields destined for the _source
* JSON column. When excludePKs is true, primary key columns are omitted from the
* column lists (for SET clauses).
* @param model *jsql.Model, data et.Json, excludePKs bool
* @return []string, []string, et.Json
**/
func sqliteColsVals(model *jsql.Model, data et.Json, excludePKs bool) (cols, vals []string, source et.Json) {
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
* sqliteReturningClause: Builds the RETURNING clause with the same shape as a SELECT without fields:
* with a SourceField the row comes back as one JSON "result" (SourceField without hidden keys plus
* the visible columns), so attributes without a declared column are returned too; without it, the
* column list. Uses command.Returns when set.
* @param command *jsql.Command
* @return string
**/
func sqliteReturningClause(command *jsql.Command) string {
	if len(command.Returns) > 0 {
		return "\nRETURNING " + strings.Join(command.Returns, ", ")
	}

	model := command.From.Model
	if model == nil {
		return "\nRETURNING *"
	}

	pairs := make([]string, 0, len(model.Columns))
	cols := make([]string, 0, len(model.Columns))
	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN || col.Name == model.SourceField || slices.Contains(model.Hiddens, col.Name) {
			continue
		}
		pairs = append(pairs, fmt.Sprintf("%s, %s", sqlitePath([]string{col.Name}), sqliteColumnJson(col.Name, col.TypeData)))
		cols = append(cols, col.Name)
	}

	if model.SourceField != "" {
		source := sqliteSourceExpr(model.SourceField, model.Hiddens)
		return fmt.Sprintf("\nRETURNING %s AS %s", sqliteSetObject(source, pairs), jsql.RESULT)
	}
	if len(cols) == 0 {
		return "\nRETURNING *"
	}
	return "\nRETURNING " + strings.Join(cols, ", ")
}

/**
* sqlitePKWhere: Builds a WHERE clause using primary key values from data.
* Returns empty string when any primary key is missing.
* @param model *jsql.Model, data et.Json
* @return string
**/
func sqlitePKWhere(model *jsql.Model, data et.Json) string {
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
* sqliteInsertSQL: Generates INSERT INTO … (cols) VALUES (vals) RETURNING …
* @param command *jsql.Command
* @return string, error
**/
func sqliteInsertSQL(command *jsql.Command) (string, error) {
	table := command.From.Name
	model := command.From.Model

	var cols, vals []string
	var source et.Json

	if model != nil {
		cols, vals, source = sqliteColsVals(model, command.New, false)
		if model.SourceField != "" && len(source) > 0 {
			cols = append(cols, model.SourceField)
			vals = append(vals, sqliteJsonValue(sqliteNestSource(source)))
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
	sb.WriteString(sqliteReturningClause(command))
	sb.WriteString(";")
	return sb.String(), nil
}

/**
* sqliteUpdateSQL: Generates UPDATE … SET … WHERE … RETURNING …
* Excludes primary key columns from SET; WHERE uses the PK values of the fetched row (command.Old),
* falling back to command.New and then to command.Conditions.
* @param command *jsql.Command
* @return string, error
**/
func sqliteUpdateSQL(command *jsql.Command) (string, error) {
	table := command.From.Name
	model := command.From.Model

	var setCols []string

	if model != nil {
		cols, vals, source := sqliteColsVals(model, command.New, true)
		for i, col := range cols {
			setCols = append(setCols, fmt.Sprintf("%s = %s", col, vals[i]))
		}
		if model.SourceField != "" && len(source) > 0 {
			setCols = append(setCols, fmt.Sprintf("%s = %s", model.SourceField, sqliteJsonSet(model.SourceField, source)))
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
		whereSQL = sqlitePKWhere(model, command.Old)
		if whereSQL == "" {
			whereSQL = sqlitePKWhere(model, command.New)
		}
	}
	if whereSQL == "" && model != nil && len(command.Conditions) > 0 {
		whereSQL = sqliteCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL == "" {
		return "", fmt.Errorf("refusing to UPDATE %s without a WHERE clause (missing primary key value in New and no Conditions set)", table)
	}
	sb.WriteString("\nWHERE " + whereSQL)

	sb.WriteString(sqliteReturningClause(command))
	sb.WriteString(";")
	return sb.String(), nil
}

/**
* sqliteDeleteSQL: Generates DELETE FROM … WHERE … RETURNING …
* WHERE uses primary key values from command.Old (the fetched row).
* @param command *jsql.Command
* @return string, error
**/
func sqliteDeleteSQL(command *jsql.Command) (string, error) {
	table := command.From.Name
	model := command.From.Model

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("DELETE FROM %s", table))

	var whereSQL string
	if model != nil && len(model.PrimaryKeys) > 0 && len(command.Old) > 0 {
		whereSQL = sqlitePKWhere(model, command.Old)
	}
	if whereSQL == "" && model != nil && len(command.Conditions) > 0 {
		whereSQL = sqliteCondsSQL(model.GetField, model.SourceField != "", command.Conditions, "")
	}
	if whereSQL == "" {
		return "", fmt.Errorf("refusing to DELETE from %s without a WHERE clause (missing primary key value in Old and no Conditions set)", table)
	}
	sb.WriteString("\nWHERE " + whereSQL)

	sb.WriteString(sqliteReturningClause(command))
	sb.WriteString(";")
	return sb.String(), nil
}

/**
* Command: Generates the SQL DML string (INSERT, UPDATE, DELETE, BULK) for the
* given Command.
* @param command *jsql.Command
* @return string, error
**/
func (s *Sqlite) Command(command *jsql.Command) (string, error) {
	switch command.Type {
	case jsql.INSERT, jsql.BULK:
		return sqliteInsertSQL(command)
	case jsql.UPDATE:
		return sqliteUpdateSQL(command)
	case jsql.DELETE:
		return sqliteDeleteSQL(command)
	default:
		return "", fmt.Errorf("unsupported command type: %s", command.Type)
	}
}
