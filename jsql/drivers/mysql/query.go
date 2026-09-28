package mysql

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
* myJoinKeyword: Maps a JoinType to its SQL keyword (FULL JOIN is rejected in Query).
* @param tp jsql.JoinType
* @return string
**/
func myJoinKeyword(tp jsql.JoinType) string {
	switch tp {
	case jsql.LEFT_JOIN:
		return "LEFT JOIN"
	case jsql.RIGHT_JOIN:
		return "RIGHT JOIN"
	default:
		return "INNER JOIN"
	}
}

/**
* mySourceColumn: Returns the SourceField reference of a field's origin.
* @param fld *jsql.Field
* @return string
**/
func mySourceColumn(fld *jsql.Field) string {
	sourceField := jsql.SOURCE
	if fld.From.Model != nil && fld.From.Model.SourceField != "" {
		sourceField = fld.From.Model.SourceField
	}
	return myColumnRef(myAlias(fld.From), sourceField)
}

/**
* myValueType: Infers the comparison type of an untyped attribute from the compared value.
* @param val any
* @return et.TypeData
**/
func myValueType(val any) et.TypeData {
	switch v := val.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return et.FLOAT
	case bool:
		return et.BOOL
	case time.Time, *time.Time:
		return et.DATETIME
	case et.BetweenValue:
		return myValueType(v.Min)
	}
	rv := reflect.ValueOf(val)
	if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) && rv.Len() > 0 {
		return myValueType(rv.Index(0).Interface())
	}
	return et.ANY
}

/**
* myScalarExpr: Resolves a field to a scalar SQL expression (WHERE, GROUP BY, ORDER BY). Attributes are
* read as text with JSON_UNQUOTE(JSON_EXTRACT(…)) and cast to the given type (booleans stay 'true'/'false').
* @param fld *jsql.Field, tp et.TypeData
* @return string
**/
func myScalarExpr(fld *jsql.Field, tp et.TypeData) string {
	if fld == nil || fld.From == nil {
		return ""
	}
	switch fld.TypeColumn {
	case jsql.COLUMN:
		return myColumnRef(myAlias(fld.From), fld.Name)
	case jsql.ATTRIB:
		expr := fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(%s, %s))", mySourceColumn(fld), myJsonPath(strings.Split(fld.Name, "->")))
		switch tp {
		case et.INT:
			return fmt.Sprintf("CAST(%s AS SIGNED)", expr)
		case et.FLOAT:
			return fmt.Sprintf("CAST(%s AS DECIMAL(38,10))", expr)
		case et.DATETIME:
			return fmt.Sprintf("CAST(%s AS DATETIME(6))", expr)
		}
		return expr
	}
	return ""
}

/**
* myJsonValueExpr: Resolves a field to a JSON value for JSON_OBJECT / JSON_SET: attributes keep the JSON
* type stored in the SourceField and booleans become true/false.
* @param fld *jsql.Field
* @return string
**/
func myJsonValueExpr(fld *jsql.Field) string {
	switch fld.TypeColumn {
	case jsql.COLUMN:
		ref := myColumnRef(myAlias(fld.From), fld.Name)
		if fld.TypeData == et.BOOL {
			return fmt.Sprintf("IF(%s IS NULL, NULL, CAST(IF(%s, 'true', 'false') AS JSON))", ref, ref)
		}
		return ref
	case jsql.ATTRIB:
		return fmt.Sprintf("JSON_EXTRACT(%s, %s)", mySourceColumn(fld), myJsonPath(strings.Split(fld.Name, "->")))
	}
	return ""
}

/**
* myAggExpr: Renders an aggregate field; SUM/AVG/MIN/MAX over attributes read them as numbers.
* @param fld *jsql.Field
* @return string, bool
**/
func myAggExpr(fld *jsql.Field) (string, bool) {
	if fld.Agg == nil || (fld.TypeColumn != jsql.COLUMN && fld.TypeColumn != jsql.ATTRIB) {
		return "", false
	}
	tp := fld.TypeData
	if fld.TypeColumn == jsql.ATTRIB && fld.Agg.Function != et.COUNT && tp != et.DATETIME {
		tp = et.FLOAT
	}
	expr := myScalarExpr(fld, tp)
	if expr == "" {
		return "", false
	}
	return myAggregate(fld.Agg.Function, expr), true
}

/**
* myCompareValue: Quotes a compared value; attribute booleans compare as 'true'/'false' text.
* @param val any, tp et.TypeData
* @return string
**/
func myCompareValue(val any, tp et.TypeData) string {
	if tp == et.BOOL {
		if b, ok := val.(bool); ok {
			return fmt.Sprintf("'%t'", b)
		}
	}
	return Quoted(et.NewValue(val))
}

/**
* myInValues: Formats a slice as an IN list with elements quoted as tp.
* @param val any, tp et.TypeData
* @return string
**/
func myInValues(val any, tp et.TypeData) string {
	rv := reflect.ValueOf(val)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return myCompareValue(val, tp)
	}
	parts := make([]string, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		parts[i] = myCompareValue(rv.Index(i).Interface(), tp)
	}
	return strings.Join(parts, ", ")
}

/**
* myColumnRefValue: In a JOIN ON (or a WHERE with several origins), resolves a string value "alias.field"
* that names a field of the query as a column reference.
* @param getField func(string) (*jsql.Field, bool), val et.Value
* @return string, bool
**/
func myColumnRefValue(getField func(string) (*jsql.Field, bool), val et.Value) (string, bool) {
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
	expr := myScalarExpr(fld, fld.TypeData)
	return expr, expr != ""
}

/**
* myFallbackField: Builds a safe reference for a field that did not resolve against the model.
* @param field, alias string
* @return string
**/
func myFallbackField(field, alias string) string {
	parts := strings.Split(field, "->")
	names := strings.Split(parts[0], ".")
	for i, name := range names {
		names[i] = myIdent(name)
	}
	if names[len(names)-1] == "``" {
		return ""
	}
	ref := strings.Join(names, ".")
	if alias != "" && len(names) == 1 {
		ref = fmt.Sprintf("%s.%s", alias, ref)
	}
	if len(parts) > 1 {
		return fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(%s, %s))", ref, myJsonPath(parts[1:]))
	}
	return ref
}

/**
* myCondExpr: Renders a single Condition. Untyped attributes are cast by the compared value; IS / IS NOT
* with a value use the NULL-safe <=>.
* @param getField func(string) (*jsql.Field, bool), cond *et.Condition, alias string, refs bool
* @return string
**/
func myCondExpr(getField func(string) (*jsql.Field, bool), cond *et.Condition, alias string, refs bool) string {
	var fieldExpr string
	tp := et.ANY
	if fld, ok := getField(cond.Field.String()); ok && fld.Agg != nil {
		fieldExpr, _ = myAggExpr(fld)
		tp = et.FLOAT
	} else if ok {
		tp = fld.TypeData
		if fld.TypeColumn == jsql.ATTRIB && (tp == et.ANY || tp == et.TEXT || tp == et.KEY) {
			if vt := myValueType(cond.Value.Value); vt != et.ANY {
				tp = vt
			}
		}
		if fld.TypeColumn == jsql.COLUMN && tp == et.BOOL {
			tp = et.INT
		}
		fieldExpr = myScalarExpr(fld, tp)
	}
	if fieldExpr == "" {
		fieldExpr = myFallbackField(cond.Field.String(), sanitizeIdent(alias))
		if fieldExpr == "" {
			return ""
		}
	}

	value := func() string {
		if refs {
			if ref, ok := myColumnRefValue(getField, cond.Value); ok {
				return ref
			}
		}
		return myCompareValue(cond.Value.Value, tp)
	}

	switch cond.Operator {
	case et.NULL:
		return fmt.Sprintf("%s IS NULL", fieldExpr)
	case et.NOT_NULL:
		return fmt.Sprintf("%s IS NOT NULL", fieldExpr)
	case et.IN:
		return fmt.Sprintf("%s IN (%s)", fieldExpr, myInValues(cond.Value.Value, tp))
	case et.NOT_IN:
		return fmt.Sprintf("%s NOT IN (%s)", fieldExpr, myInValues(cond.Value.Value, tp))
	case et.BETWEEN, et.NOT_BETWEEN:
		bv, ok := cond.Value.Value.(et.BetweenValue)
		if !ok {
			return ""
		}
		op := "BETWEEN"
		if cond.Operator == et.NOT_BETWEEN {
			op = "NOT BETWEEN"
		}
		return fmt.Sprintf("%s %s %s AND %s", fieldExpr, op, myCompareValue(bv.Min, tp), myCompareValue(bv.Max, tp))
	case et.LIKE:
		return fmt.Sprintf("LOWER(%s) LIKE LOWER(%s)", fieldExpr, value())
	case et.IS:
		if cond.Value.Value == nil {
			return fmt.Sprintf("%s IS NULL", fieldExpr)
		}
		return fmt.Sprintf("%s <=> %s", fieldExpr, value())
	case et.IS_NOT:
		if cond.Value.Value == nil {
			return fmt.Sprintf("%s IS NOT NULL", fieldExpr)
		}
		return fmt.Sprintf("NOT (%s <=> %s)", fieldExpr, value())
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
* myConds: Renders a Condition slice joined by AND/OR connectors.
* @param getField func(string) (*jsql.Field, bool), conds []*et.Condition, alias string, refs bool
* @return string
**/
func myConds(getField func(string) (*jsql.Field, bool), conds []*et.Condition, alias string, refs bool) string {
	var parts []string
	first := true
	for _, cond := range conds {
		expr := myCondExpr(getField, cond, alias, refs)
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
* mySourceExpr: Returns the SourceField without the hidden keys; a NULL source counts as {}.
* @param source string, hiddens []string
* @return string
**/
func mySourceExpr(source string, hiddens []string) string {
	expr := fmt.Sprintf("COALESCE(%s, JSON_OBJECT())", source)
	if len(hiddens) == 0 {
		return expr
	}
	paths := make([]string, len(hiddens))
	for i, h := range hiddens {
		paths[i] = myJsonPath([]string{h})
	}
	return fmt.Sprintf("JSON_REMOVE(%s, %s)", expr, strings.Join(paths, ", "))
}

/**
* mySetObject: Builds JSON_SET(base, path, value, …) in chunks of 50 pairs; JSON_SET keeps null values.
* @param base string, pairs []string ("path, value")
* @return string
**/
func mySetObject(base string, pairs []string) string {
	const maxPairs = 50
	expr := base
	for i := 0; i < len(pairs); i += maxPairs {
		end := min(i+maxPairs, len(pairs))
		expr = fmt.Sprintf("JSON_SET(%s,\n%s\n)", expr, strings.Join(pairs[i:end], ",\n"))
	}
	return expr
}

/**
* mySelectExpr: Resolves an explicit select entry as a JSON_OBJECT pair ("'as', expr"). Relations
* (details, masters, rollups, calcs) are registered in the query and produce no SQL.
* @param query *jsql.Query, field string
* @return string, bool
**/
func mySelectExpr(query *jsql.Query, field string) (string, bool) {
	fld, ok := query.GetField(field)
	if !ok {
		return "", false
	}
	pair := func(expr string) (string, bool) {
		return fmt.Sprintf("%s, %s", myQuoteKey(fld.As), expr), true
	}
	if fld.Agg != nil {
		expr, ok := myAggExpr(fld)
		if !ok {
			return "", false
		}
		return pair(expr)
	}
	switch fld.TypeColumn {
	case jsql.COLUMN, jsql.ATTRIB:
		// A grouped query selects the expression it groups by; ANY_VALUE tells MySQL (only_full_group_by)
		// that it is single-valued per group, which it cannot see through the JSON functions.
		if len(query.GroupsBy) > 0 {
			return pair(fmt.Sprintf("ANY_VALUE(%s)", myScalarExpr(fld, fld.TypeData)))
		}
		return pair(myJsonValueExpr(fld))
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
* myRowObject: Builds the JSON of a whole row: the SourceField (without hidden keys) with every visible
* column set on it, or a JSON_OBJECT of the columns when the model has no SourceField.
* @param from *jsql.From, extraHiddens []string
* @return string
**/
func myRowObject(from *jsql.From, extraHiddens []string) string {
	model := from.Model
	alias := myAlias(from)
	hiddens := append(slices.Clone(extraHiddens), model.Hiddens...)
	pairs := make([]string, 0, len(model.Columns))
	objectPairs := make([]string, 0, len(model.Columns))
	for _, col := range model.Columns {
		if col.TypeColumn != jsql.COLUMN || col.Name == model.SourceField || slices.Contains(hiddens, col.Name) {
			continue
		}
		fld := &jsql.Field{Field: et.Field{Name: col.Name, As: col.Name}, TypeColumn: jsql.COLUMN, TypeData: col.TypeData, From: from}
		value := myJsonValueExpr(fld)
		pairs = append(pairs, fmt.Sprintf("%s, %s", myJsonPath([]string{col.Name}), value))
		objectPairs = append(objectPairs, fmt.Sprintf("%s, %s", myQuoteKey(col.Name), value))
	}
	if model.SourceField != "" {
		return mySetObject(mySourceExpr(myColumnRef(alias, model.SourceField), hiddens), pairs)
	}
	return fmt.Sprintf("JSON_OBJECT(%s)", strings.Join(objectPairs, ", "))
}

/**
* mySelects: Generates the SELECT list. The row always comes back as one JSON "result": the requested
* fields, or with no fields the whole row (SourceField plus columns, without hidden ones).
* @param query *jsql.Query
* @return string
**/
func mySelects(query *jsql.Query) string {
	if len(query.Selects) == 0 {
		if len(query.Froms) == 1 {
			return fmt.Sprintf("%s AS %s", myRowObject(query.Froms[0], query.Hiddens), myIdent(jsql.RESULT))
		}
		pairs := make([]string, 0)
		for _, from := range query.Froms {
			for _, col := range from.Model.Columns {
				if col.TypeColumn != jsql.COLUMN || col.Name == from.Model.SourceField || slices.Contains(from.Model.Hiddens, col.Name) || slices.Contains(query.Hiddens, col.Name) {
					continue
				}
				if expr, ok := mySelectExpr(query, fmt.Sprintf("%s.%s", from.As, col.Name)); ok {
					pairs = append(pairs, expr)
				}
			}
		}
		return fmt.Sprintf("JSON_OBJECT(%s) AS %s", strings.Join(pairs, ",\n"), myIdent(jsql.RESULT))
	}

	pairs := make([]string, 0, len(query.Selects))
	for _, field := range query.Selects {
		if slices.Contains(query.Hiddens, field) || field == jsql.SOURCE {
			continue
		}
		if expr, ok := mySelectExpr(query, field); ok {
			pairs = append(pairs, expr)
		}
	}
	return fmt.Sprintf("JSON_OBJECT(\n%s\n) AS %s", strings.Join(pairs, ",\n"), myIdent(jsql.RESULT))
}

/**
* myFromClause: Returns "FROM table AS alias, …".
* @param query *jsql.Query
* @return string
**/
func myFromClause(query *jsql.Query) string {
	refs := make([]string, 0, len(query.Froms))
	for _, from := range query.Froms {
		ref := myFromRef(from)
		if alias := myAlias(from); alias != "" {
			ref = fmt.Sprintf("%s AS %s", ref, alias)
		}
		refs = append(refs, ref)
	}
	return "\nFROM " + strings.Join(refs, ",\n")
}

/**
* myOrderExpr: Resolves an ORDER BY entry: a selected alias first (e.g. of an aggregate), then any field.
* @param query *jsql.Query, name string
* @return string, bool
**/
func myOrderExpr(query *jsql.Query, name string) (string, bool) {
	fld, ok := query.GetSelectField(name)
	if !ok {
		fld, ok = query.GetField(name)
	}
	if !ok {
		return "", false
	}
	if fld.Agg != nil {
		return myAggExpr(fld)
	}
	expr := myScalarExpr(fld, fld.TypeData)
	return expr, expr != ""
}

/**
* Query: Generates the SELECT for the given Query. FULL JOIN returns ErrFullJoin (MySQL has none);
* EXISTS returns 'true'/'false' in "exists" and COUNT the number of rows in "count".
* @param query *jsql.Query
* @return string, error
**/
func (s *Mysql) Query(query *jsql.Query) (string, error) {
	if len(query.Froms) == 0 {
		return "", fmt.Errorf("query has no FROM source")
	}
	for _, join := range query.Joins {
		if join.Type == jsql.FULL_JOIN {
			return "", ErrFullJoin
		}
	}

	primary := query.Froms[0]
	alias := myAlias(primary)

	var sb strings.Builder
	switch {
	case query.IsExists:
		sb.WriteString("SELECT 1")
	case query.IsCount:
		sb.WriteString("SELECT COUNT(*) AS `count`")
	default:
		sb.WriteString("SELECT\n")
		sb.WriteString(mySelects(query))
	}
	sb.WriteString(myFromClause(query))

	for _, join := range query.Joins {
		ref := myFromRef(join.To)
		if a := myAlias(join.To); a != "" {
			ref = fmt.Sprintf("%s AS %s", ref, a)
		}
		sb.WriteString(fmt.Sprintf("\n%s %s", myJoinKeyword(join.Type), ref))
		if on := myConds(query.GetField, join.Condition, myAlias(join.To), true); on != "" {
			sb.WriteString("\n  ON " + on)
		}
	}

	// With several origins the tables are related in the WHERE, where "alias.field" values are columns.
	if where := myConds(query.GetField, query.Conditions, alias, len(query.Froms) > 1); where != "" {
		sb.WriteString("\nWHERE " + where)
	}

	if len(query.GroupsBy) > 0 && !query.IsExists {
		exprs := make([]string, 0, len(query.GroupsBy))
		for _, name := range query.GroupsBy {
			if fld, ok := query.GetField(name); ok {
				exprs = append(exprs, myScalarExpr(fld, fld.TypeData))
			}
		}
		if len(exprs) > 0 {
			sb.WriteString("\nGROUP BY " + strings.Join(exprs, ", "))
		}
	}

	if having := myConds(query.GetField, query.Havings, alias, false); having != "" && !query.IsExists {
		sb.WriteString("\nHAVING " + having)
	}

	if query.IsExists {
		return fmt.Sprintf("SELECT CASE WHEN EXISTS(%s) THEN 'true' ELSE 'false' END AS `exists`", sb.String()), nil
	}
	if query.IsCount {
		return sb.String(), nil
	}

	if len(query.OrdersBy) > 0 {
		parts := make([]string, 0, len(query.OrdersBy))
		for _, idx := range query.OrdersBy {
			expr, ok := myOrderExpr(query, idx.Name)
			if !ok {
				continue
			}
			dir := "ASC"
			if !idx.Sorted {
				dir = "DESC"
			}
			parts = append(parts, fmt.Sprintf("%s %s", expr, dir))
		}
		if len(parts) > 0 {
			sb.WriteString("\nORDER BY " + strings.Join(parts, ", "))
		}
	}

	if query.Rows > 0 {
		sb.WriteString(fmt.Sprintf("\nLIMIT %d", query.Rows))
	} else if query.Offset > 0 {
		sb.WriteString("\nLIMIT 18446744073709551615")
	}
	if query.Offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", query.Offset))
	}
	return sb.String(), nil
}
