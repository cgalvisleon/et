package oracle

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
* oraJoinKeyword: Maps a JoinType to its SQL keyword.
* @param tp jsql.JoinType
* @return string
**/
func oraJoinKeyword(tp jsql.JoinType) string {
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
* oraSourceColumn: Returns the SourceField reference (alias."_source") of a field's origin.
* @param fld *jsql.Field
* @return string
**/
func oraSourceColumn(fld *jsql.Field) string {
	sourceField := jsql.SOURCE
	if fld.From.Model != nil && fld.From.Model.SourceField != "" {
		sourceField = fld.From.Model.SourceField
	}
	return oraColumnRef(oraAlias(fld.From), sourceField)
}

/**
* oraReturning: Returns the RETURNING clause of JSON_VALUE for a type ("" for text).
* @param tp et.TypeData
* @return string
**/
func oraReturning(tp et.TypeData) string {
	switch tp {
	case et.INT, et.FLOAT:
		return " RETURNING NUMBER"
	case et.DATETIME:
		return " RETURNING TIMESTAMP"
	default:
		return ""
	}
}

/**
* oraValueType: Infers the comparison type of an untyped attribute from the compared value.
* @param val any
* @return et.TypeData
**/
func oraValueType(val any) et.TypeData {
	switch v := val.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return et.FLOAT
	case bool:
		return et.BOOL
	case time.Time, *time.Time:
		return et.DATETIME
	case et.BetweenValue:
		return oraValueType(v.Min)
	}
	rv := reflect.ValueOf(val)
	if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) && rv.Len() > 0 {
		return oraValueType(rv.Index(0).Interface())
	}
	return et.ANY
}

/**
* oraScalarExpr: Resolves a field to a scalar SQL expression (WHERE, GROUP BY, ORDER BY, plain SELECT).
* ATTRIB fields read the SourceField with JSON_VALUE, returning the given type
* (booleans stay 'true'/'false' text).
* @param fld *jsql.Field, tp et.TypeData
* @return string
**/
func oraScalarExpr(fld *jsql.Field, tp et.TypeData) string {
	if fld == nil || fld.From == nil {
		return ""
	}
	switch fld.TypeColumn {
	case jsql.COLUMN:
		return oraColumnRef(oraAlias(fld.From), fld.Name)
	case jsql.ATTRIB:
		path := oraJsonPath(strings.Split(fld.Name, "->"))
		return fmt.Sprintf("JSON_VALUE(%s, %s%s)", oraSourceColumn(fld), path, oraReturning(tp))
	}
	return ""
}

/**
* oraJsonValueExpr: Resolves a field to the value part of a JSON_OBJECT pair ("expr [FORMAT JSON]").
* Attributes keep the JSON type stored in the SourceField: the value is read with an array wrapper
* and unwrapped, which works for scalars, objects and arrays from Oracle 19c on.
* @param fld *jsql.Field
* @return string
**/
func oraJsonValueExpr(fld *jsql.Field) string {
	switch fld.TypeColumn {
	case jsql.COLUMN:
		ref := oraColumnRef(oraAlias(fld.From), fld.Name)
		if fld.TypeData == et.BOOL {
			return fmt.Sprintf("(CASE %s WHEN 1 THEN 'true' WHEN 0 THEN 'false' END) FORMAT JSON", ref)
		}
		if isJsonType(fld.TypeData) {
			return ref + " FORMAT JSON"
		}
		return ref
	case jsql.ATTRIB:
		path := oraJsonPath(strings.Split(fld.Name, "->"))
		wrapped := fmt.Sprintf("JSON_QUERY(%s, %s RETURNING CLOB WITH ARRAY WRAPPER)", oraSourceColumn(fld), path)
		return fmt.Sprintf("SUBSTR(%s, 2, LENGTH(%s) - 2) FORMAT JSON", wrapped, wrapped)
	}
	return ""
}

/**
* oraAggExpr: Renders an aggregate field; SUM/AVG/MIN/MAX over attributes read them as NUMBER.
* @param fld *jsql.Field
* @return string, bool
**/
func oraAggExpr(fld *jsql.Field) (string, bool) {
	if fld.Agg == nil || (fld.TypeColumn != jsql.COLUMN && fld.TypeColumn != jsql.ATTRIB) {
		return "", false
	}
	tp := fld.TypeData
	if fld.TypeColumn == jsql.ATTRIB && fld.Agg.Function != et.COUNT && tp != et.DATETIME {
		tp = et.FLOAT
	}
	expr := oraScalarExpr(fld, tp)
	if expr == "" {
		return "", false
	}
	return oraAggregate(fld.Agg.Function, expr), true
}

/**
* findField: Returns the Field for field if it resolves against the query origins.
* @param query *jsql.Query, field string
* @return *jsql.Field, bool
**/
func findField(query *jsql.Query, field string) (*jsql.Field, bool) {
	if query == nil {
		return nil, false
	}
	return query.GetField(field)
}

/**
* oraInValues: Formats a Go slice as a comma-separated SQL IN list, with elements quoted as tp.
* @param val any, tp et.TypeData
* @return string
**/
func oraInValues(val any, tp et.TypeData) string {
	rv := reflect.ValueOf(val)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return oraCompareValue(val, tp)
	}
	parts := make([]string, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		parts[i] = oraCompareValue(rv.Index(i).Interface(), tp)
	}
	return strings.Join(parts, ", ")
}

/**
* oraCompareValue: Quotes a compared value for a field of type tp; attribute booleans compare
* as 'true'/'false' text because JSON_VALUE returns them that way.
* @param val any, tp et.TypeData
* @return string
**/
func oraCompareValue(val any, tp et.TypeData) string {
	if tp == et.BOOL {
		if b, ok := val.(bool); ok {
			return fmt.Sprintf("'%t'", b)
		}
	}
	return Quoted(et.NewValue(val))
}

/**
* oraJoinColumnRef: In a JOIN ON condition, resolves a string value "alias.field" that names a field
* of the query origins as a column reference instead of a literal.
* @param getField func(string) (*jsql.Field, bool), val et.Value
* @return string, bool
**/
func oraJoinColumnRef(getField func(string) (*jsql.Field, bool), val et.Value) (string, bool) {
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
	expr := oraScalarExpr(fld, fld.TypeData)
	return expr, expr != ""
}

/**
* oraFallbackField: Builds a safe reference for a field that did not resolve against the model.
* @param field, alias string
* @return string
**/
func oraFallbackField(field, alias string) string {
	parts := strings.Split(field, "->")
	names := strings.Split(parts[0], ".")
	for i, name := range names {
		names[i] = oraIdent(name)
	}
	if names[len(names)-1] == `""` {
		return ""
	}
	ref := strings.Join(names, ".")
	if alias != "" && len(names) == 1 {
		ref = fmt.Sprintf("%s.%s", alias, ref)
	}
	if len(parts) > 1 {
		return fmt.Sprintf("JSON_VALUE(%s, %s)", ref, oraJsonPath(parts[1:]))
	}
	return ref
}

/**
* oraCondExpr: Renders a single Condition as a SQL fragment.
* Untyped attributes are read with the type of the compared value (NUMBER, TIMESTAMP, or
* 'true'/'false' text for booleans).
* @param getField func(string) (*jsql.Field, bool), cond *et.Condition, alias string, isJoin bool
* @return string
**/
func oraCondExpr(getField func(string) (*jsql.Field, bool), cond *et.Condition, alias string, isJoin bool) string {
	var fieldExpr string
	tp := et.ANY
	if fld, ok := getField(cond.Field.String()); ok && fld.Agg != nil {
		fieldExpr, _ = oraAggExpr(fld)
		tp = et.FLOAT
	} else if ok {
		tp = fld.TypeData
		if fld.TypeColumn == jsql.ATTRIB && (tp == et.ANY || tp == et.TEXT || tp == et.KEY) {
			if vt := oraValueType(cond.Value.Value); vt != et.ANY {
				tp = vt
			}
		}
		if fld.TypeColumn == jsql.COLUMN && tp == et.BOOL {
			tp = et.INT
		}
		fieldExpr = oraScalarExpr(fld, tp)
	}
	if fieldExpr == "" {
		fieldExpr = oraFallbackField(cond.Field.String(), sanitizeIdent(alias))
		if fieldExpr == "" {
			return ""
		}
	}

	value := func() string {
		if isJoin {
			if ref, ok := oraJoinColumnRef(getField, cond.Value); ok {
				return ref
			}
		}
		return oraCompareValue(cond.Value.Value, tp)
	}

	switch cond.Operator {
	case et.NULL:
		return fmt.Sprintf("%s IS NULL", fieldExpr)
	case et.NOT_NULL:
		return fmt.Sprintf("%s IS NOT NULL", fieldExpr)
	case et.IN:
		return fmt.Sprintf("%s IN (%s)", fieldExpr, oraInValues(cond.Value.Value, tp))
	case et.NOT_IN:
		return fmt.Sprintf("%s NOT IN (%s)", fieldExpr, oraInValues(cond.Value.Value, tp))
	case et.BETWEEN, et.NOT_BETWEEN:
		bv, ok := cond.Value.Value.(et.BetweenValue)
		if !ok {
			return ""
		}
		op := "BETWEEN"
		if cond.Operator == et.NOT_BETWEEN {
			op = "NOT BETWEEN"
		}
		return fmt.Sprintf("%s %s %s AND %s", fieldExpr, op, oraCompareValue(bv.Min, tp), oraCompareValue(bv.Max, tp))
	case et.LIKE:
		return fmt.Sprintf("UPPER(%s) LIKE UPPER(%s)", fieldExpr, value())
	case et.IS:
		if cond.Value.Value == nil {
			return fmt.Sprintf("%s IS NULL", fieldExpr)
		}
		return fmt.Sprintf("%s = %s", fieldExpr, value())
	case et.IS_NOT:
		if cond.Value.Value == nil {
			return fmt.Sprintf("%s IS NOT NULL", fieldExpr)
		}
		return fmt.Sprintf("%s != %s", fieldExpr, value())
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
* oraConds: Renders a Condition slice as a SQL clause body joined by AND/OR connectors.
* @param getField func(string) (*jsql.Field, bool), conds []*et.Condition, alias string, isJoin bool
* @return string
**/
func oraConds(getField func(string) (*jsql.Field, bool), conds []*et.Condition, alias string, isJoin bool) string {
	var parts []string
	first := true
	for _, cond := range conds {
		expr := oraCondExpr(getField, cond, alias, isJoin)
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
* oraSourceExpr: Returns the SourceField as a CLOB without the hidden keys; a NULL source counts as '{}'.
* Hidden keys are removed with a merge patch that sets them to null.
* @param source string, hiddens []string
* @return string
**/
func oraSourceExpr(source string, hiddens []string) string {
	expr := fmt.Sprintf("NVL(%s, TO_CLOB('{}'))", source)
	if len(hiddens) == 0 {
		return expr
	}
	patch := et.Json{}
	for _, h := range hiddens {
		patch[h] = nil
	}
	return fmt.Sprintf("JSON_MERGEPATCH(%s, %s RETURNING CLOB)", expr, oraQuoteJson(patch))
}

/**
* oraMergeObject: Builds JSON_MERGEPATCH(source, JSON_OBJECT(pairs...)), splitting pairs into chunks
* of 50. Merge patch drops keys whose value is null, so null columns are omitted from the result.
* @param source string, pairs []string ("'key' VALUE expr")
* @return string
**/
func oraMergeObject(source string, pairs []string) string {
	const maxPairs = 50
	objects := make([]string, 0)
	for i := 0; i < len(pairs); i += maxPairs {
		end := min(i+maxPairs, len(pairs))
		objects = append(objects, fmt.Sprintf("JSON_OBJECT(\n%s\nRETURNING CLOB)", strings.Join(pairs[i:end], ",\n")))
	}

	expr := source
	for _, obj := range objects {
		if expr == "" {
			expr = obj
			continue
		}
		expr = fmt.Sprintf("JSON_MERGEPATCH(%s, %s RETURNING CLOB)", expr, obj)
	}
	if expr == "" {
		return "TO_CLOB('{}')"
	}
	return expr
}

/**
* oraSelectExpr: Resolves an explicit select entry. With a SourceField it returns a JSON_OBJECT pair
* ("'as' VALUE expr"); without it, "expr AS "as"". Relations (details, masters, rollups, calcs)
* are registered in the query and produce no SQL.
* @param query *jsql.Query, field string
* @return string, bool
**/
func oraSelectExpr(query *jsql.Query, field string) (string, bool) {
	fld, ok := findField(query, field)
	if !ok {
		return "", false
	}

	pair := func(expr string) (string, bool) {
		if query.UseSourceField {
			return fmt.Sprintf("%s VALUE %s", oraQuoteKey(fld.As), expr), true
		}
		expr = strings.TrimSuffix(expr, " FORMAT JSON")
		return fmt.Sprintf("%s AS %s", expr, oraIdent(fld.As)), true
	}

	if fld.Agg != nil {
		expr, ok := oraAggExpr(fld)
		if !ok {
			return "", false
		}
		return pair(expr)
	}

	switch fld.TypeColumn {
	case jsql.COLUMN, jsql.ATTRIB:
		// A grouped query must select the same expression it groups by.
		if !query.UseSourceField || len(query.GroupsBy) > 0 {
			return pair(oraScalarExpr(fld, fld.TypeData))
		}
		return pair(oraJsonValueExpr(fld))
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
			query.Masters[fld.Name] = &jsql.QueryDetail{To: master.To, Keys: master.Keys, Select: []string{}, Page: fld.Page, Rows: query.MaxRows}
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
* oraSelects: Generates the SELECT list. With a SourceField the row is one JSON "result":
* JSON_OBJECT of the requested fields, or, with no fields, the SourceField (without hidden keys)
* merged with all visible columns.
* @param query *jsql.Query
* @return string
**/
func oraSelects(query *jsql.Query) string {
	exprs := make([]string, 0)
	if len(query.Selects) > 0 {
		for _, field := range query.Selects {
			if slices.Contains(query.Hiddens, field) || field == jsql.SOURCE {
				continue
			}
			if expr, ok := oraSelectExpr(query, field); ok {
				exprs = append(exprs, expr)
			}
		}
	} else {
		for _, from := range query.Froms {
			model := from.Model
			for _, col := range model.Columns {
				if col.TypeColumn != jsql.COLUMN || col.Name == model.SourceField {
					continue
				}
				if slices.Contains(query.Hiddens, col.Name) || slices.Contains(model.Hiddens, col.Name) {
					continue
				}
				name := col.Name
				if len(query.Froms) > 1 {
					name = fmt.Sprintf("%s.%s", from.As, col.Name)
				}
				if expr, ok := oraSelectExpr(query, name); ok {
					exprs = append(exprs, expr)
				}
			}
		}
	}

	if !query.UseSourceField {
		if len(exprs) == 0 {
			return "*"
		}
		return strings.Join(exprs, ",\n")
	}

	source := ""
	if len(query.Selects) == 0 && len(query.Froms) > 0 {
		from := query.Froms[0]
		sourceField := jsql.SOURCE
		hiddens := slices.Clone(query.Hiddens)
		if from.Model != nil {
			if from.Model.SourceField != "" {
				sourceField = from.Model.SourceField
			}
			hiddens = append(hiddens, from.Model.Hiddens...)
		}
		source = oraSourceExpr(oraColumnRef(oraAlias(from), sourceField), hiddens)
	}
	return fmt.Sprintf("%s AS %s", oraMergeObject(source, exprs), oraIdent(jsql.RESULT))
}

/**
* oraFrom: Generates the FROM clause (Oracle table aliases take no AS).
* @param query *jsql.Query
* @return string
**/
func oraFrom(query *jsql.Query) string {
	refs := make([]string, 0, len(query.Froms))
	for _, from := range query.Froms {
		ref := oraFromRef(from)
		if alias := oraAlias(from); alias != "" {
			ref = fmt.Sprintf("%s %s", ref, alias)
		}
		refs = append(refs, ref)
	}
	return "\nFROM " + strings.Join(refs, ",\n")
}

/**
* Query: Generates the SQL SELECT for the given Query descriptor. Paging uses
* OFFSET … ROWS FETCH NEXT … ROWS ONLY (12c+); EXISTS returns 'true'/'false' in "exists"
* and COUNT the number of rows in "count".
* @param query *jsql.Query
* @return string, error
**/
func (s *Oracle) Query(query *jsql.Query) (string, error) {
	if len(query.Froms) == 0 {
		return "", fmt.Errorf("query has no FROM source")
	}

	primary := query.Froms[0]
	alias := oraAlias(primary)

	var sb strings.Builder
	switch {
	case query.IsExists:
		sb.WriteString("SELECT 1")
	case query.IsCount:
		sb.WriteString(fmt.Sprintf("SELECT COUNT(*) AS %s", oraIdent("count")))
	default:
		sb.WriteString("SELECT\n")
		sb.WriteString(oraSelects(query))
	}

	sb.WriteString(oraFrom(query))

	for _, join := range query.Joins {
		ref := oraFromRef(join.To)
		if a := oraAlias(join.To); a != "" {
			ref = fmt.Sprintf("%s %s", ref, a)
		}
		sb.WriteString(fmt.Sprintf("\n%s %s", oraJoinKeyword(join.Type), ref))
		if on := oraConds(query.GetField, join.Condition, oraAlias(join.To), true); on != "" {
			sb.WriteString("\n  ON " + on)
		}
	}

	if where := oraConds(query.GetField, query.Conditions, alias, false); where != "" {
		sb.WriteString("\nWHERE " + where)
	}

	if len(query.GroupsBy) > 0 && !query.IsExists {
		exprs := make([]string, 0, len(query.GroupsBy))
		for _, name := range query.GroupsBy {
			if fld, ok := query.GetField(name); ok {
				exprs = append(exprs, oraScalarExpr(fld, fld.TypeData))
			}
		}
		if len(exprs) > 0 {
			sb.WriteString("\nGROUP BY " + strings.Join(exprs, ", "))
		}
	}

	if having := oraConds(query.GetField, query.Havings, alias, false); having != "" && !query.IsExists {
		sb.WriteString("\nHAVING " + having)
	}

	if query.IsExists {
		return fmt.Sprintf(`SELECT CASE WHEN EXISTS(%s) THEN 'true' ELSE 'false' END AS %s FROM DUAL`, sb.String(), oraIdent("exists")), nil
	}
	if query.IsCount {
		return sb.String(), nil
	}

	if len(query.OrdersBy) > 0 {
		parts := make([]string, 0, len(query.OrdersBy))
		for _, idx := range query.OrdersBy {
			fld, ok := query.GetField(idx.Name)
			if !ok {
				continue
			}
			dir := "ASC"
			if !idx.Sorted {
				dir = "DESC"
			}
			parts = append(parts, fmt.Sprintf("%s %s", oraScalarExpr(fld, fld.TypeData), dir))
		}
		if len(parts) > 0 {
			sb.WriteString("\nORDER BY " + strings.Join(parts, ", "))
		}
	}

	if query.Offset > 0 {
		sb.WriteString(fmt.Sprintf("\nOFFSET %d ROWS", query.Offset))
	}
	if query.Rows > 0 {
		sb.WriteString(fmt.Sprintf("\nFETCH NEXT %d ROWS ONLY", query.Rows))
	}

	return sb.String(), nil
}
