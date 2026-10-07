package mssql

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* msJoinKeyword: Maps a JoinType to its SQL keyword.
* @param tp jsql.JoinType
* @return string
**/
func msJoinKeyword(tp jsql.JoinType) string {
	switch tp {
	case jsql.LEFT_JOIN:
		return "LEFT JOIN"
	case jsql.RIGHT_JOIN:
		return "RIGHT JOIN"
	case jsql.FULL_JOIN:
		return "FULL JOIN"
	default:
		return "INNER JOIN"
	}
}

/**
* msSourceColumn: Returns the SourceField reference of a field's origin.
* @param fld *jsql.Field
* @return string
**/
func msSourceColumn(fld *jsql.Field) string {
	sourceField := jsql.SOURCE
	if fld.From.Model != nil && fld.From.Model.SourceField != "" {
		sourceField = fld.From.Model.SourceField
	}
	return msColumnRef(msAlias(fld.From), sourceField)
}

/**
* msFragment: Returns the JSON text of a SQL value, used to build the row JSON as text: strings are quoted
* and escaped, numbers and booleans written as such, times in ISO 8601, JSON columns embedded, NULL as null.
* @param expr string, tp et.TypeData
* @return string
**/
func msFragment(expr string, tp et.TypeData) string {
	switch tp {
	case et.BOOL:
		return fmt.Sprintf("CASE WHEN %s IS NULL THEN 'null' WHEN %s = 1 THEN 'true' ELSE 'false' END", expr, expr)
	case et.INT:
		return fmt.Sprintf("COALESCE(CONVERT(NVARCHAR(40), %s), 'null')", expr)
	case et.FLOAT:
		return fmt.Sprintf("COALESCE(CONVERT(NVARCHAR(60), %s), 'null')", expr)
	case et.DATETIME:
		return fmt.Sprintf("CASE WHEN %s IS NULL THEN 'null' ELSE CONCAT('\"', CONVERT(NVARCHAR(40), %s, 127), '\"') END", expr, expr)
	}
	if isJsonType(tp) {
		return fmt.Sprintf("COALESCE(%s, 'null')", expr)
	}
	return fmt.Sprintf("CASE WHEN %s IS NULL THEN 'null' ELSE CONCAT('\"', STRING_ESCAPE(CAST(%s AS NVARCHAR(MAX)), 'json'), '\"') END", expr, expr)
}

/**
* msFloatColumn: Formats a FLOAT column with 17 significant digits (CONVERT style 3), so the JSON keeps
* the full precision.
* @param expr string
* @return string
**/
func msFloatColumn(expr string) string {
	return fmt.Sprintf("COALESCE(CONVERT(NVARCHAR(60), %s, 3), 'null')", expr)
}

/**
* msAttribFragment: Returns the JSON text of an attribute of the SourceField keeping its JSON type: OPENJSON
* returns the raw value and its type, and strings are quoted again. A missing attribute is null.
* @param source string, parts []string
* @return string
**/
func msAttribFragment(source string, parts []string) string {
	parent := msJsonPathText(parts[:len(parts)-1])
	leaf := parts[len(parts)-1]
	return fmt.Sprintf("COALESCE((SELECT TOP 1 CASE [type] WHEN 1 THEN CONCAT('\"', STRING_ESCAPE([value], 'json'), '\"') WHEN 0 THEN 'null' ELSE [value] END FROM OPENJSON(%s, %s) WHERE [key] = %s), 'null')",
		source, msQuoteText(parent), msQuoteText(leaf))
}

/**
* msValueType: Infers the comparison type of an untyped attribute from the compared value.
* @param val any
* @return et.TypeData
**/
func msValueType(val any) et.TypeData {
	switch v := val.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return et.FLOAT
	case bool:
		return et.BOOL
	case time.Time, *time.Time:
		return et.DATETIME
	case et.BetweenValue:
		return msValueType(v.Min)
	}
	rv := reflect.ValueOf(val)
	if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) && rv.Len() > 0 {
		return msValueType(rv.Index(0).Interface())
	}
	return et.ANY
}

/**
* msScalarExpr: Resolves a field to a scalar SQL expression (WHERE, GROUP BY, ORDER BY). Attributes are
* read with JSON_VALUE and converted to the given type with TRY_CAST (booleans stay 'true'/'false').
* @param fld *jsql.Field, tp et.TypeData
* @return string
**/
func msScalarExpr(fld *jsql.Field, tp et.TypeData) string {
	if fld == nil || fld.From == nil {
		return ""
	}
	switch fld.TypeColumn {
	case jsql.COLUMN:
		return msColumnRef(msAlias(fld.From), fld.Name)
	case jsql.ATTRIB:
		expr := fmt.Sprintf("JSON_VALUE(%s, %s)", msSourceColumn(fld), msJsonPath(strings.Split(fld.Name, "->")))
		switch tp {
		case et.INT:
			return fmt.Sprintf("TRY_CAST(%s AS BIGINT)", expr)
		case et.FLOAT:
			return fmt.Sprintf("TRY_CAST(%s AS DECIMAL(38,10))", expr)
		case et.DATETIME:
			return fmt.Sprintf("TRY_CAST(%s AS DATETIME2)", expr)
		}
		return expr
	}
	return ""
}

/**
* msValueFragment: Returns the JSON text of a field for the row JSON: columns by their type, attributes
* keeping their stored JSON type.
* @param fld *jsql.Field
* @return string
**/
func msValueFragment(fld *jsql.Field) string {
	switch fld.TypeColumn {
	case jsql.COLUMN:
		ref := msColumnRef(msAlias(fld.From), fld.Name)
		if fld.TypeData == et.FLOAT {
			return msFloatColumn(ref)
		}
		return msFragment(ref, fld.TypeData)
	case jsql.ATTRIB:
		return msAttribFragment(msSourceColumn(fld), strings.Split(fld.Name, "->"))
	}
	return "'null'"
}

/**
* msAggExpr: Renders an aggregate field and the type of its result; SUM/AVG/MIN/MAX over attributes read
* them as numbers.
* @param fld *jsql.Field
* @return string, et.TypeData, bool
**/
func msAggExpr(fld *jsql.Field) (string, et.TypeData, bool) {
	if fld.Agg == nil || (fld.TypeColumn != jsql.COLUMN && fld.TypeColumn != jsql.ATTRIB) {
		return "", et.ANY, false
	}
	tp := fld.TypeData
	if fld.TypeColumn == jsql.ATTRIB && fld.Agg.Function != et.COUNT && tp != et.DATETIME {
		tp = et.FLOAT
	}
	expr := msScalarExpr(fld, tp)
	if expr == "" {
		return "", et.ANY, false
	}
	resultType := tp
	switch fld.Agg.Function {
	case et.COUNT:
		resultType = et.INT
	case et.SUM, et.AVG:
		resultType = et.FLOAT
	}
	return msAggregate(fld.Agg.Function, expr), resultType, true
}

/**
* msCompareValue: Quotes a compared value; attribute booleans compare as 'true'/'false' text.
* @param val any, tp et.TypeData
* @return string
**/
func msCompareValue(val any, tp et.TypeData) string {
	if tp == et.BOOL {
		if b, ok := val.(bool); ok {
			return fmt.Sprintf("'%t'", b)
		}
	}
	return Quoted(et.NewValue(val))
}

/**
* msInValues: Formats a slice as an IN list with elements quoted as tp.
* @param val any, tp et.TypeData
* @return string
**/
func msInValues(val any, tp et.TypeData) string {
	rv := reflect.ValueOf(val)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return msCompareValue(val, tp)
	}
	parts := make([]string, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		parts[i] = msCompareValue(rv.Index(i).Interface(), tp)
	}
	return strings.Join(parts, ", ")
}

/**
* msColumnRefValue: In a JOIN ON (or a WHERE with several origins), resolves a string value "alias.field"
* that names a field of the query as a column reference.
* @param getField func(string) (*jsql.Field, bool), val et.Value
* @return string, bool
**/
func msColumnRefValue(getField func(string) (*jsql.Field, bool), val et.Value) (string, bool) {
	str, ok := val.Value.(string)
	if !ok || !strings.Contains(str, ".") {
		return "", false
	}
	def, ok := et.ToField(str)
	if !ok || def.Source == "" || def.Agg != nil {
		return "", false
	}
	fld, ok := getField(str)
	if !ok || fld.From == nil || (fld.From.As != def.Source && fld.From.Name != def.Source) {
		return "", false
	}
	expr := msScalarExpr(fld, fld.TypeData)
	return expr, expr != ""
}

/**
* msFallbackField: Builds a safe reference for a field that did not resolve against the model.
* @param field, alias string
* @return string
**/
func msFallbackField(field, alias string) string {
	parts := strings.Split(field, "->")
	names := strings.Split(parts[0], ".")
	for i, name := range names {
		names[i] = msIdent(name)
	}
	if names[len(names)-1] == "[]" {
		return ""
	}
	ref := strings.Join(names, ".")
	if alias != "" && len(names) == 1 {
		ref = fmt.Sprintf("%s.%s", alias, ref)
	}
	if len(parts) > 1 {
		return fmt.Sprintf("JSON_VALUE(%s, %s)", ref, msJsonPath(parts[1:]))
	}
	return ref
}

/**
* msCondExpr: Renders a single Condition. Untyped attributes are cast by the compared value; IS / IS NOT
* with a value use IS [NOT] DISTINCT FROM.
* @param getField func(string) (*jsql.Field, bool), cond *et.Condition, alias string, refs bool
* @return string
**/
func msCondExpr(getField func(string) (*jsql.Field, bool), cond *et.Condition, alias string, refs bool) string {
	var fieldExpr string
	tp := et.ANY
	if fld, ok := getField(cond.Field.String()); ok && fld.Agg != nil {
		fieldExpr, _, _ = msAggExpr(fld)
		tp = et.FLOAT
	} else if ok {
		tp = fld.TypeData
		if fld.TypeColumn == jsql.ATTRIB && (tp == et.ANY || tp == et.TEXT || tp == et.KEY) {
			if vt := msValueType(cond.Value.Value); vt != et.ANY {
				tp = vt
			}
		}
		if fld.TypeColumn == jsql.COLUMN && tp == et.BOOL {
			tp = et.INT
		}
		fieldExpr = msScalarExpr(fld, tp)
	}
	if fieldExpr == "" {
		fieldExpr = msFallbackField(cond.Field.String(), sanitizeIdent(alias))
		if fieldExpr == "" {
			return ""
		}
	}

	value := func() string {
		if refs {
			if ref, ok := msColumnRefValue(getField, cond.Value); ok {
				return ref
			}
		}
		return msCompareValue(cond.Value.Value, tp)
	}

	switch cond.Operator {
	case et.NULL:
		return fmt.Sprintf("%s IS NULL", fieldExpr)
	case et.NOT_NULL:
		return fmt.Sprintf("%s IS NOT NULL", fieldExpr)
	case et.IN:
		return fmt.Sprintf("%s IN (%s)", fieldExpr, msInValues(cond.Value.Value, tp))
	case et.NOT_IN:
		return fmt.Sprintf("%s NOT IN (%s)", fieldExpr, msInValues(cond.Value.Value, tp))
	case et.BETWEEN, et.NOT_BETWEEN:
		bv, ok := cond.Value.Value.(et.BetweenValue)
		if !ok {
			return ""
		}
		op := "BETWEEN"
		if cond.Operator == et.NOT_BETWEEN {
			op = "NOT BETWEEN"
		}
		return fmt.Sprintf("%s %s %s AND %s", fieldExpr, op, msCompareValue(bv.Min, tp), msCompareValue(bv.Max, tp))
	case et.LIKE:
		return fmt.Sprintf("LOWER(%s) LIKE LOWER(%s)", fieldExpr, value())
	case et.IS:
		if cond.Value.Value == nil {
			return fmt.Sprintf("%s IS NULL", fieldExpr)
		}
		return fmt.Sprintf("%s IS NOT DISTINCT FROM %s", fieldExpr, value())
	case et.IS_NOT:
		if cond.Value.Value == nil {
			return fmt.Sprintf("%s IS NOT NULL", fieldExpr)
		}
		return fmt.Sprintf("%s IS DISTINCT FROM %s", fieldExpr, value())
	case et.NEG:
		return fmt.Sprintf("%s != %s", fieldExpr, value())
	case et.LESS:
		return fmt.Sprintf("%s < %s", fieldExpr, value())
	case et.LESS_EQ:
		return fmt.Sprintf("%s <= %s", fieldExpr, value())
	case et.MORE:
		return fmt.Sprintf("%s > %s", fieldExpr, value())
	case et.MORE_EQ:
		return fmt.Sprintf("%s >= %s", fieldExpr, value())
	default:
		return fmt.Sprintf("%s = %s", fieldExpr, value())
	}
}

/**
* msConds: Renders a Condition slice joined by AND/OR connectors.
* @param getField func(string) (*jsql.Field, bool), conds []*et.Condition, alias string, refs bool
* @return string
**/
func msConds(getField func(string) (*jsql.Field, bool), conds []*et.Condition, alias string, refs bool) string {
	var parts []string
	first := true
	for _, cond := range conds {
		expr := msCondExpr(getField, cond, alias, refs)
		if expr == "" {
			continue
		}
		if first || cond.Connector == et.NaC {
			parts = append(parts, expr)
			first = false
		} else if cond.Connector == et.OR {
			parts = append(parts, "OR "+expr)
		} else {
			parts = append(parts, "AND "+expr)
		}
	}
	return strings.Join(parts, "\n  ")
}

/**
* msConcat: Builds CONCAT(args…), nesting calls so none exceeds the 254-argument limit.
* @param args []string
* @return string
**/
func msConcat(args []string) string {
	const maxArgs = 200
	if len(args) == 0 {
		return "''"
	}
	if len(args) == 1 {
		return fmt.Sprintf("CONCAT(%s, '')", args[0])
	}
	if len(args) <= maxArgs {
		return fmt.Sprintf("CONCAT(%s)", strings.Join(args, ", "))
	}
	groups := make([]string, 0)
	for i := 0; i < len(args); i += maxArgs {
		end := min(i+maxArgs, len(args))
		groups = append(groups, msConcat(args[i:end]))
	}
	return msConcat(groups)
}

/**
* msPairs: Returns the CONCAT arguments of the "key":value pairs, separated by commas.
* @param pairs []string (each "N'\"key\":', fragment")
* @return []string
**/
func msPairs(pairs []string) []string {
	args := make([]string, 0, len(pairs)*2)
	for i, pair := range pairs {
		if i > 0 {
			args = append(args, "','")
		}
		args = append(args, pair)
	}
	return args
}

/**
* msObject: Builds the JSON text of an object from its pairs.
* @param pairs []string
* @return string
**/
func msObject(pairs []string) string {
	args := append([]string{"'{'"}, msPairs(pairs)...)
	args = append(args, "'}'")
	return msConcat(args)
}

/**
* msSourceExpr: Returns the SourceField as JSON text without the hidden keys (JSON_MODIFY with NULL removes
* them); a NULL source counts as {}.
* @param source string, hiddens []string
* @return string
**/
func msSourceExpr(source string, hiddens []string) string {
	expr := fmt.Sprintf("COALESCE(%s, N'{}')", source)
	for _, h := range hiddens {
		expr = fmt.Sprintf("JSON_MODIFY(%s, %s, NULL)", expr, msJsonPath([]string{h}))
	}
	return expr
}

/**
* msMergeObject: Builds the JSON text of the SourceField with the pairs added after its own keys. When a
* key is repeated, the later one (the column) is the one a JSON parser keeps.
* @param source string, pairs []string
* @return string
**/
func msMergeObject(source string, pairs []string) string {
	inner := fmt.Sprintf("SUBSTRING(%s, 2, LEN(%s) - 2)", source, source)
	args := []string{"'{'", inner}
	if len(pairs) > 0 {
		args = append(args, fmt.Sprintf("CASE WHEN LEN(%s) > 2 THEN ',' ELSE '' END", source))
		args = append(args, msPairs(pairs)...)
	}
	args = append(args, "'}'")
	return msConcat(args)
}

/**
* msSelectExpr: Resolves an explicit select entry as a JSON pair ("N'\"as\":', fragment"). Relations are
* registered in the query and produce no SQL.
* @param query *jsql.Query, field string
* @return string, bool
**/
func msSelectExpr(query *jsql.Query, field string) (string, bool) {
	fld, ok := query.GetField(field)
	if !ok {
		return "", false
	}
	pair := func(fragment string) (string, bool) {
		return fmt.Sprintf("%s, %s", msKeyPrefix(fld.As), fragment), true
	}
	if fld.Agg != nil {
		expr, tp, ok := msAggExpr(fld)
		if !ok {
			return "", false
		}
		return pair(msFragment(expr, tp))
	}
	switch fld.TypeColumn {
	case jsql.COLUMN, jsql.ATTRIB:
		// A grouped query selects the expression it groups by.
		if len(query.GroupsBy) > 0 {
			return pair(msFragment(msScalarExpr(fld, fld.TypeData), fld.TypeData))
		}
		return pair(msValueFragment(fld))
	}

	model := fld.From.Model
	if model == nil {
		return "", false
	}
	switch fld.TypeColumn {
	case jsql.DETAIL:
		if detail, ok := model.Details[fld.Name]; ok {
			query.Details[fld.Name] = &jsql.QueryDetail{To: detail.To, Keys: detail.Keys, Select: detail.Select, Page: fld.Page, Rows: detail.Rows}
		}
	case jsql.MASTER:
		if master, ok := model.Masters[fld.Name]; ok {
			rows := master.Rows
			if rows <= 0 {
				rows = query.MaxRows
			}
			query.Masters[fld.Name] = &jsql.QueryDetail{To: master.To, Bridge: master.Bridge, ToKeys: master.ToKeys, Keys: master.Keys, Select: master.Select, Page: fld.Page, Rows: rows}
		}
	case jsql.ROLLUP:
		if rollup, ok := model.Rollups[fld.Name]; ok {
			query.Rollups[fld.Name] = &jsql.QueryRollups{To: rollup.To, Keys: rollup.Keys, Select: rollup.Select, Operation: rollup.Operation}
		}
	case jsql.CALC:
		query.Calcs[fld.Name] = &jsql.Calc{Model: model, Module: ""}
	case jsql.CALCFUNC:
		if calc, ok := model.GetCalcFunc(fld.Name); ok {
			query.CalcFuns[fld.Name] = calc
		}
	}
	return "", false
}

/**
* msRowObject: Builds the JSON text of a whole row: the SourceField (without hidden keys) with every
* visible column added, or an object of the columns when the model has no SourceField.
* @param from *jsql.From, extraHiddens []string
* @return string
**/
func msRowObject(from *jsql.From, extraHiddens []string) string {
	model := from.Model
	hiddens := append(slices.Clone(extraHiddens), model.Hiddens...)
	pairs := make([]string, 0, len(model.Columns))
	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN || col.Name == model.SourceField || slices.Contains(hiddens, col.Name) {
			continue
		}
		fld := &jsql.Field{Field: et.Field{Name: col.Name, As: col.Name}, TypeColumn: jsql.COLUMN, TypeData: col.TypeData, From: from}
		pairs = append(pairs, fmt.Sprintf("%s, %s", msKeyPrefix(col.Name), msValueFragment(fld)))
	}
	if model.SourceField != "" {
		return msMergeObject(msSourceExpr(msColumnRef(msAlias(from), model.SourceField), hiddens), pairs)
	}
	return msObject(pairs)
}

/**
* msSelects: Generates the SELECT list: the row always comes back as JSON text in "result".
* @param query *jsql.Query
* @return string
**/
func msSelects(query *jsql.Query) string {
	if len(query.Selects) == 0 {
		if len(query.Froms) == 1 {
			return fmt.Sprintf("%s AS %s", msRowObject(query.Froms[0], query.Hiddens), msIdent(jsql.RESULT))
		}
		pairs := make([]string, 0)
		for _, from := range query.Froms {
			for _, col := range from.Model.Columns {
				if col.TypeColumn != jsql.COLUMN || col.Name == from.Model.SourceField || slices.Contains(from.Model.Hiddens, col.Name) || slices.Contains(query.Hiddens, col.Name) {
					continue
				}
				if expr, ok := msSelectExpr(query, fmt.Sprintf("%s.%s", from.As, col.Name)); ok {
					pairs = append(pairs, expr)
				}
			}
		}
		return fmt.Sprintf("%s AS %s", msObject(pairs), msIdent(jsql.RESULT))
	}

	// "*" son las columnas de los from (StarFields); se le suman los demás campos de selects
	pairs := make([]string, 0, len(query.Selects))
	add := func(field string) {
		if expr, ok := msSelectExpr(query, field); ok {
			pairs = append(pairs, expr)
		}
	}
	for _, field := range query.Selects {
		if field == jsql.STAR {
			for _, name := range query.StarFields() {
				add(name)
			}
			continue
		}
		if slices.Contains(query.Hiddens, field) || field == jsql.SOURCE {
			continue
		}
		add(field)
	}

	// Con "*" y campo fuente, los atributos del primer from y encima los campos
	if from := query.Froms[0]; query.HasStar() && from.Model != nil && from.Model.SourceField != "" {
		source := msSourceExpr(msColumnRef(msAlias(from), from.Model.SourceField), query.SourceHiddens(from))
		return fmt.Sprintf("%s AS %s", msMergeObject(source, pairs), msIdent(jsql.RESULT))
	}
	return fmt.Sprintf("%s AS %s", msObject(pairs), msIdent(jsql.RESULT))
}

/**
* msFromClause: Returns "FROM table AS alias, …".
* @param query *jsql.Query
* @return string
**/
func msFromClause(query *jsql.Query) string {
	refs := make([]string, 0, len(query.Froms))
	for _, from := range query.Froms {
		ref := msFromRef(from)
		if alias := msAlias(from); alias != "" {
			ref = fmt.Sprintf("%s AS %s", ref, alias)
		}
		refs = append(refs, ref)
	}
	return "\nFROM " + strings.Join(refs, ",\n")
}

/**
* msOrderExpr: Resolves an ORDER BY entry: a selected alias first (e.g. of an aggregate), then any field.
* @param query *jsql.Query, name string
* @return string, bool
**/
func msOrderExpr(query *jsql.Query, name string) (string, bool) {
	fld, ok := query.GetSelectField(name)
	if !ok {
		fld, ok = query.GetField(name)
	}
	if !ok {
		return "", false
	}
	if fld.Agg != nil {
		expr, _, ok := msAggExpr(fld)
		return expr, ok
	}
	expr := msScalarExpr(fld, fld.TypeData)
	return expr, expr != ""
}

/**
* Query: Generates the SELECT for the given Query. Paging uses OFFSET … FETCH, which needs an ORDER BY
* (ORDER BY (SELECT NULL) when the query has none); EXISTS returns 'true'/'false' in "exists".
* @param query *jsql.Query
* @return string, error
**/
func (s *Mssql) Query(query *jsql.Query, timeout ...time.Duration) (string, error) {
	if len(query.Froms) == 0 {
		return "", fmt.Errorf("query has no FROM source")
	}
	primary := query.Froms[0]
	alias := msAlias(primary)

	var sb strings.Builder
	switch {
	case query.IsExists:
		sb.WriteString("SELECT 1")
	case query.IsCount:
		sb.WriteString("SELECT COUNT(*) AS [count]")
	default:
		sb.WriteString("SELECT\n")
		sb.WriteString(msSelects(query))
	}
	sb.WriteString(msFromClause(query))

	for _, join := range query.Joins {
		ref := msFromRef(join.To)
		if a := msAlias(join.To); a != "" {
			ref = fmt.Sprintf("%s AS %s", ref, a)
		}
		sb.WriteString(fmt.Sprintf("\n%s %s", msJoinKeyword(join.Type), ref))
		if on := msConds(query.GetField, join.Condition, msAlias(join.To), true); on != "" {
			sb.WriteString("\n  ON " + on)
		}
	}

	// With several origins the tables are related in the WHERE, where "alias.field" values are columns.
	if where := msConds(query.GetField, query.Conditions, alias, len(query.Froms) > 1); where != "" {
		sb.WriteString("\nWHERE " + where)
	}

	if len(query.GroupsBy) > 0 && !query.IsExists {
		exprs := make([]string, 0, len(query.GroupsBy))
		for _, name := range query.GroupsBy {
			if fld, ok := query.GetField(name); ok {
				exprs = append(exprs, msScalarExpr(fld, fld.TypeData))
			}
		}
		if len(exprs) > 0 {
			sb.WriteString("\nGROUP BY " + strings.Join(exprs, ", "))
		}
	}

	if having := msConds(query.GetField, query.Havings, alias, false); having != "" && !query.IsExists {
		sb.WriteString("\nHAVING " + having)
	}

	if query.IsExists {
		return fmt.Sprintf("SELECT CASE WHEN EXISTS(%s) THEN 'true' ELSE 'false' END AS [exists]", sb.String()), nil
	}
	if query.IsCount {
		return sb.String(), nil
	}

	parts := make([]string, 0, len(query.OrdersBy))
	for _, idx := range query.OrdersBy {
		expr, ok := msOrderExpr(query, idx.Name)
		if !ok {
			continue
		}
		dir := "ASC"
		if !idx.Sorted {
			dir = "DESC"
		}
		parts = append(parts, fmt.Sprintf("%s %s", expr, dir))
	}
	paged := query.Rows > 0 || query.Offset > 0
	if len(parts) > 0 {
		sb.WriteString("\nORDER BY " + strings.Join(parts, ", "))
	} else if paged {
		sb.WriteString("\nORDER BY (SELECT NULL)")
	}
	if paged {
		sb.WriteString(fmt.Sprintf("\nOFFSET %d ROWS", max(query.Offset, 0)))
		if query.Rows > 0 {
			sb.WriteString(fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", query.Rows))
		}
	}
	return sb.String(), nil
}
