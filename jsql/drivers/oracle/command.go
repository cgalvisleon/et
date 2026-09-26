package oracle

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* oraNestSource: Expands '->' keys of source into nested objects ({"a->b": 1} → {"a": {"b": 1}}).
* @param source et.Json
* @return et.Json
**/
func oraNestSource(source et.Json) et.Json {
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
* oraColsVals: Separates data into sorted parallel (quoted column names, quoted values) slices,
* quoting each value for its column type, and a source et.Json for ATTRIB fields.
* When excludePKs is true, primary key columns are omitted (for SET clauses).
* @param model *jsql.Model, data et.Json, excludePKs bool
* @return []string, []string, et.Json
**/
func oraColsVals(model *jsql.Model, data et.Json, excludePKs bool) (cols, vals []string, source et.Json) {
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
		cols[i] = oraIdent(name)
		vals[i] = QuotedAs(columns[name].TypeData, data[name])
	}
	return
}

/**
* oraRowResult: Builds the expression returned for each affected row, with the same shape as a SELECT
* without fields: with a SourceField, one JSON "result" (SourceField without hidden keys merged with the
* visible columns); without it, the visible columns. Uses command.Returns when set.
* @param command *jsql.Command
* @return string
**/
func oraRowResult(command *jsql.Command) string {
	if len(command.Returns) > 0 {
		return strings.Join(command.Returns, ", ")
	}

	model := command.From.Model
	if model == nil {
		return "*"
	}

	pairs := make([]string, 0, len(model.Columns))
	cols := make([]string, 0, len(model.Columns))
	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN || col.Name == model.SourceField || slices.Contains(model.Hiddens, col.Name) {
			continue
		}
		fld := &jsql.Field{Field: et.Field{Name: col.Name, As: col.Name}, TypeColumn: jsql.COLUMN, TypeData: col.TypeData, From: command.From}
		pairs = append(pairs, fmt.Sprintf("%s VALUE %s", oraQuoteKey(col.Name), oraJsonValueExpr(fld)))
		cols = append(cols, fmt.Sprintf("%s AS %s", oraIdent(col.Name), oraIdent(col.Name)))
	}

	if model.SourceField != "" {
		source := oraSourceExpr(oraIdent(model.SourceField), model.Hiddens)
		return fmt.Sprintf("%s AS %s", oraMergeObject(source, pairs), oraIdent(jsql.RESULT))
	}
	if len(cols) == 0 {
		return "*"
	}
	return strings.Join(cols, ", ")
}

/**
* oraPKWhere: Builds a WHERE clause from the primary key values in data.
* Returns empty string when any primary key is missing.
* @param model *jsql.Model, data et.Json
* @return string
**/
func oraPKWhere(model *jsql.Model, data et.Json) string {
	conds := make([]string, 0, len(model.PrimaryKeys))
	for _, pk := range model.PrimaryKeys {
		val, ok := data[pk.Name]
		if !ok {
			return ""
		}
		conds = append(conds, fmt.Sprintf("%s = %s", oraIdent(pk.Name), QuotedAs(columnType(model, pk.Name), val)))
	}
	return strings.Join(conds, " AND ")
}

/**
* oraWhere: Returns the WHERE body for UPDATE/DELETE: the primary key of the first row in rows
* that has it, otherwise the command Conditions.
* @param command *jsql.Command, rows ...et.Json
* @return string, error
**/
func oraWhere(command *jsql.Command, rows ...et.Json) (string, error) {
	model := command.From.Model
	if model != nil && len(model.PrimaryKeys) > 0 {
		for _, row := range rows {
			if where := oraPKWhere(model, row); where != "" {
				return where, nil
			}
		}
	}
	if model != nil && len(command.Conditions) > 0 {
		if where := oraConds(model.GetField, command.Conditions, "", false); where != "" {
			return where, nil
		}
	}
	return "", fmt.Errorf("refusing to %s %s without a WHERE clause (missing primary key value and no Conditions set)", strings.ToUpper(string(command.Type)), oraFromRef(command.From))
}

/**
* oraBlock: Wraps a DML statement in a PL/SQL block that collects the ROWIDs it touches and
* returns those rows with DBMS_SQL.RETURN_RESULT, since Oracle has no RETURNING result set.
* @param table, dml, result string
* @return string
**/
func oraBlock(table, dml, result string) string {
	return fmt.Sprintf(`DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  %s
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT %s FROM %s WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;`, dml, result, table)
}

/**
* oraInsertSQL: Generates the INSERT block; attributes are stored as one nested JSON in the SourceField.
* @param command *jsql.Command
* @return string, error
**/
func oraInsertSQL(command *jsql.Command) (string, error) {
	table := oraFromRef(command.From)
	model := command.From.Model

	var cols, vals []string
	if model != nil {
		var source et.Json
		cols, vals, source = oraColsVals(model, command.New, false)
		if model.SourceField != "" && len(source) > 0 {
			cols = append(cols, oraIdent(model.SourceField))
			vals = append(vals, oraQuoteJson(oraNestSource(source)))
		}
	} else {
		keys := make([]string, 0, len(command.New))
		for k := range command.New {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			cols = append(cols, oraIdent(k))
			vals = append(vals, Quoted(et.NewValue(command.New[k])))
		}
	}

	if len(cols) == 0 {
		return "", fmt.Errorf("no columns to insert into %s", table)
	}

	dml := fmt.Sprintf("INSERT INTO %s\n  (%s)\n  VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(vals, ", "))
	return oraBlock(table, dml, oraRowResult(command)), nil
}

/**
* oraUpdateSQL: Generates the UPDATE block. Attributes are deep-merged into the SourceField with
* JSON_MERGEPATCH, so the other keys and nested siblings are kept; a null value removes its key.
* @param command *jsql.Command
* @return string, error
**/
func oraUpdateSQL(command *jsql.Command) (string, error) {
	table := oraFromRef(command.From)
	model := command.From.Model

	var sets []string
	if model != nil {
		cols, vals, source := oraColsVals(model, command.New, true)
		for i, col := range cols {
			sets = append(sets, fmt.Sprintf("%s = %s", col, vals[i]))
		}
		if model.SourceField != "" && len(source) > 0 {
			src := oraIdent(model.SourceField)
			sets = append(sets, fmt.Sprintf("%s = JSON_MERGEPATCH(NVL(%s, TO_CLOB('{}')), %s RETURNING CLOB)", src, src, oraQuoteJson(oraNestSource(source))))
		}
	} else {
		keys := make([]string, 0, len(command.New))
		for k := range command.New {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sets = append(sets, fmt.Sprintf("%s = %s", oraIdent(k), Quoted(et.NewValue(command.New[k]))))
		}
	}

	if len(sets) == 0 {
		return "", fmt.Errorf("no columns to update in %s", table)
	}

	where, err := oraWhere(command, command.Old, command.New)
	if err != nil {
		return "", err
	}

	dml := fmt.Sprintf("UPDATE %s\n  SET %s\n  WHERE %s", table, strings.Join(sets, ",\n    "), where)
	return oraBlock(table, dml, oraRowResult(command)), nil
}

/**
* oraDeleteSQL: Generates the DELETE block. The deleted rows are read first (the cursor keeps the
* snapshot taken when it is opened) and returned after the DELETE.
* @param command *jsql.Command
* @return string, error
**/
func oraDeleteSQL(command *jsql.Command) (string, error) {
	table := oraFromRef(command.From)
	where, err := oraWhere(command, command.Old)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`DECLARE
  c SYS_REFCURSOR;
BEGIN
  OPEN c FOR SELECT %s FROM %s WHERE %s;
  DELETE FROM %s WHERE %s;
  DBMS_SQL.RETURN_RESULT(c);
END;`, oraRowResult(command), table, where, table, where), nil
}

/**
* Command: Generates the PL/SQL block for the given Command (INSERT, BULK, UPDATE, DELETE); the block
* runs the DML and returns the affected rows as an implicit result, so the command gets its RETURNING data.
* @param command *jsql.Command
* @return string, error
**/
func (s *Oracle) Command(command *jsql.Command) (string, error) {
	switch command.Type {
	case jsql.INSERT, jsql.BULK:
		return oraInsertSQL(command)
	case jsql.UPDATE:
		return oraUpdateSQL(command)
	case jsql.DELETE:
		return oraDeleteSQL(command)
	default:
		return "", fmt.Errorf("unsupported command type: %s", command.Type)
	}
}
