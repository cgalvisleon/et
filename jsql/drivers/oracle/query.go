package oracle

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* orFromRef: Returns the "schema"."name" or "name" table reference for FROM/JOIN clauses.
* @param f *jsql.From
* @return string
**/
func orFromRef(f *jsql.From) string {
	if f.Schema != "" {
		return fmt.Sprintf("\"%s\".\"%s\"", f.Schema, f.Name)
	}
	return fmt.Sprintf("\"%s\"", f.Name)
}

/**
* orJoinKeyword: Maps a JoinType to its Oracle SQL keyword.
* Oracle supports FULL OUTER JOIN; FULL JOIN shorthand is expanded.
* @param tp jsql.JoinType
* @return string
**/
func orJoinKeyword(tp jsql.JoinType) string {
	switch tp {
	case jsql.LEFT_JOIN:
		return "LEFT JOIN"
	case jsql.RIGHT_JOIN:
		return "RIGHT JOIN"
	case jsql.FULL_JOIN:
		return "FULL OUTER JOIN"
	default:
		return "INNER JOIN"
	}
}

/**
* orFieldExpr: Resolves a logical field name to an Oracle SQL expression.
* COLUMN fields are double-quote-qualified; ATTRIB fields use JSON_VALUE with RETURNING.
* @param field *jsql.Field, useSource bool
* @return string
**/
func orFieldExpr(field *jsql.Field, useSource bool) string {
	if field == nil || field.From == nil {
		return ""
	}
	alias := field.From.As
	if field.TypeColumn == jsql.COLUMN {
		if alias == "" {
			return fmt.Sprintf("\"%s\"", field.Name)
		}
		return fmt.Sprintf("\"%s\".\"%s\"", alias, field.Name)
	}
	if field.TypeColumn == jsql.ATTRIB {
		if useSource {
			sourceField := jsql.SOURCE
			if field.From.Model != nil && field.From.Model.SourceField != "" {
				sourceField = field.From.Model.SourceField
			}
			var col string
			if alias != "" {
				col = fmt.Sprintf("\"%s\".\"%s\"", alias, sourceField)
			} else {
				col = fmt.Sprintf("\"%s\"", sourceField)
			}
			return orJsonValue(col, field.Name, field.TypeData)
		}
		if alias == "" {
			return fmt.Sprintf("\"%s\"", field.Name)
		}
		return fmt.Sprintf("\"%s\".\"%s\"", alias, field.Name)
	}
	return ""
}

/**
* findField: Returns the Field for field if it is an explicitly-defined field in the query.
* @param query *jsql.Query
* @param field string
* @return *jsql.Field, bool
**/
func findField(query *jsql.Query, field string) (*jsql.Field, bool) {
	if query == nil {
		return nil, false
	}
	result, ok := query.GetField(field)
	if !ok {
		return nil, false
	}
	return result, true
}

/**
* orInValues: Formats a Go slice as a comma-separated SQL IN-list using reflection.
* @param val any
* @return string
**/
func orInValues(val any) string {
	rv := reflect.ValueOf(val)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return fmt.Sprintf("%v", jsql.Quoted(val))
	}
	parts := make([]string, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		parts[i] = fmt.Sprintf("%v", jsql.Quoted(rv.Index(i).Interface()))
	}
	return strings.Join(parts, ", ")
}

/**
* orCondExpr: Renders a single Condition as an Oracle SQL fragment.
* ATTRIB fields resolve via JSON_VALUE; LIKE replaces ILIKE (Oracle uses NLS_COMP/NLS_SORT).
* @param getField func(string) (*jsql.Field, bool), useSourceField bool, cond *et.Condition, alias string
* @return string
**/
func orCondExpr(getField func(string) (*jsql.Field, bool), useSourceField bool, cond *et.Condition, alias string) string {
	var fieldExpr string
	if fld, ok := getField(cond.Field); ok {
		fieldExpr = orFieldExpr(fld, useSourceField)
	}
	if fieldExpr == "" {
		f := cond.Field
		if strings.Contains(f, "->") {
			segs := strings.SplitN(f, "->", 2)
			col := segs[0]
			if alias != "" && !strings.Contains(col, ".") {
				col = fmt.Sprintf("\"%s\".\"%s\"", alias, col)
			} else {
				col = fmt.Sprintf("\"%s\"", col)
			}
			jsonPath := "'$." + strings.ReplaceAll(segs[1], "->", ".") + "'"
			fieldExpr = fmt.Sprintf("JSON_VALUE(%s, %s)", col, jsonPath)
		} else {
			if alias != "" && !strings.Contains(f, ".") {
				f = fmt.Sprintf("\"%s\".\"%s\"", alias, f)
			}
			fieldExpr = f
		}
	}
	switch cond.Operator {
	case et.NULL:
		return fmt.Sprintf("%s IS NULL", fieldExpr)
	case et.NOT_NULL:
		return fmt.Sprintf("%s IS NOT NULL", fieldExpr)
	case et.IN:
		return fmt.Sprintf("%s IN (%s)", fieldExpr, orInValues(cond.Value))
	case et.NOT_IN:
		return fmt.Sprintf("%s NOT IN (%s)", fieldExpr, orInValues(cond.Value))
	case et.BETWEEN:
		bv, ok := cond.Value.(et.BetweenValue)
		if !ok {
			return ""
		}
		return fmt.Sprintf("%s BETWEEN %v AND %v", fieldExpr, jsql.Quoted(bv.Min), jsql.Quoted(bv.Max))
	case et.NOT_BETWEEN:
		bv, ok := cond.Value.(et.BetweenValue)
		if !ok {
			return ""
		}
		return fmt.Sprintf("%s NOT BETWEEN %v AND %v", fieldExpr, jsql.Quoted(bv.Min), jsql.Quoted(bv.Max))
	case et.LIKE:
		return fmt.Sprintf("%s LIKE %v", fieldExpr, jsql.Quoted(cond.Value))
	case et.IS:
		return fmt.Sprintf("%s IS %v", fieldExpr, jsql.Quoted(cond.Value))
	case et.IS_NOT:
		return fmt.Sprintf("%s IS NOT %v", fieldExpr, jsql.Quoted(cond.Value))
	case et.NEG:
		return fmt.Sprintf("%s != %v", fieldExpr, jsql.Quoted(cond.Value))
	case et.LESS:
		return fmt.Sprintf("%s < %v", fieldExpr, jsql.Quoted(cond.Value))
	case et.LESS_EQ:
		return fmt.Sprintf("%s <= %v", fieldExpr, jsql.Quoted(cond.Value))
	case et.MORE:
		return fmt.Sprintf("%s > %v", fieldExpr, jsql.Quoted(cond.Value))
	case et.MORE_EQ:
		return fmt.Sprintf("%s >= %v", fieldExpr, jsql.Quoted(cond.Value))
	default:
		return fmt.Sprintf("%s = %v", fieldExpr, jsql.Quoted(cond.Value))
	}
}

/**
* orCondsSQL: Renders a Condition slice as an Oracle SQL clause body joined by AND/OR connectors.
* @param getField func(string) (*jsql.Field, bool), useSourceField bool, conds []*et.Condition, alias string
* @return string
**/
func orCondsSQL(getField func(string) (*jsql.Field, bool), useSourceField bool, conds []*et.Condition, alias string) string {
	var parts []string
	first := true
	for _, cond := range conds {
		expr := orCondExpr(getField, useSourceField, cond, alias)
		if expr == "" {
			continue
		}
		if first || cond.Connector == et.NaC {
			parts = append(parts, expr)
			first = false
		} else if cond.Connector == et.And {
			parts = append(parts, "AND "+expr)
		} else {
			parts = append(parts, "OR "+expr)
		}
	}
	return strings.Join(parts, "\n  ")
}

/**
* orSelectExpr: Resolves an explicit field name from query.Selects into an Oracle SQL expression.
* ATTRIB columns use JSON_VALUE with RETURNING; identifiers are double-quote-qualified.
* @param query *jsql.Query, field string
* @return string, bool
**/
func orSelectExpr(query *jsql.Query, field string) (string, bool) {
	fld, ok := findField(query, field)
	if !ok {
		return "", false
	}
	alias := fld.From.As
	if fld.TypeColumn == jsql.COLUMN {
		if query.UseSourceField {
			return fmt.Sprintf("'%s', \"%s\".\"%s\"", fld.As, alias, fld.Name), true
		}
		return fmt.Sprintf("\"%s\".\"%s\" AS \"%s\"", alias, fld.Name, fld.As), true
	}
	if fld.TypeColumn == jsql.ATTRIB {
		sourceField := jsql.SOURCE
		if fld.From.Model != nil && fld.From.Model.SourceField != "" {
			sourceField = fld.From.Model.SourceField
		}
		var col string
		if alias != "" {
			col = fmt.Sprintf("\"%s\".\"%s\"", alias, sourceField)
		} else {
			col = fmt.Sprintf("\"%s\"", sourceField)
		}
		expr := orJsonValue(col, fld.Name, fld.TypeData)
		if query.UseSourceField {
			return fmt.Sprintf("'%s', %s", fld.As, expr), true
		}
		return fmt.Sprintf("%s AS \"%s\"", expr, fld.As), true
	}
	if fld.TypeColumn == jsql.DETAIL {
		if fld.From == nil {
			return "", false
		}
		if fld.From.Model == nil {
			return "", false
		}
		detail, ok := fld.From.Model.Details[fld.Name]
		if !ok {
			return "", false
		}
		query.Details[fld.Name] = detail
	}
	if fld.TypeColumn == jsql.ROLLUP {
		if fld.From == nil {
			return "", false
		}
		if fld.From.Model == nil {
			return "", false
		}
		rollup, ok := fld.From.Model.Rollups[fld.Name]
		if !ok {
			return "", false
		}
		query.Rollups[fld.Name] = rollup
	}
	if fld.TypeColumn == jsql.RELATION {
		if fld.From == nil {
			return "", false
		}
		if fld.From.Model == nil {
			return "", false
		}
		relation, ok := fld.From.Model.Relations[fld.Name]
		if !ok {
			return "", false
		}
		query.Relations[fld.Name] = relation
	}

	return "", false
}

/**
* orSelects: Generates the SQL SELECT expression list for the given Query descriptor.
* In UseSourceField mode builds a JSON_OBJECT for explicit selects or merges _source
* with column values via JSON_MERGEPATCH for auto-selects.
* @param query *jsql.Query
* @return []string
**/
func orSelects(query *jsql.Query) []string {
	var selectExprs []string

	if len(query.Selects) > 0 {
		for _, field := range query.Selects {
			if slices.Contains(query.Hiddens, field) {
				continue
			}
			selectExpr, ok := orSelectExpr(query, field)
			if !ok {
				continue
			}
			selectExprs = append(selectExprs, selectExpr)
		}
	}

	if query.UseSourceField {
		if len(query.Selects) > 0 {
			inner := strings.Join(selectExprs, ", ")
			return []string{fmt.Sprintf("JSON_OBJECT(%s RETURNING CLOB) AS \"%s\"", inner, jsql.RESULT)}
		}
		var result []string
		for _, from := range query.Froms {
			model := from.Model
			var pairs []string
			for _, col := range model.Columns {
				if slices.Contains(query.Hiddens, col.Name) {
					continue
				}
				if slices.Contains(model.Hiddens, col.Name) {
					continue
				}
				if col.Name == jsql.SOURCE {
					continue
				}
				selectExpr, ok := orSelectExpr(query, col.Name)
				if !ok {
					continue
				}
				pairs = append(pairs, selectExpr)
			}
			src := fmt.Sprintf("\"%s\".\"%s\"", from.As, jsql.SOURCE)
			if len(pairs) > 0 {
				inner := strings.Join(pairs, ", ")
				result = append(result, fmt.Sprintf(
					"JSON_MERGEPATCH(%s, JSON_OBJECT(%s RETURNING CLOB) RETURNING CLOB)",
					src, inner))
			} else {
				result = append(result, src)
			}
		}
		return []string{fmt.Sprintf("%s AS \"%s\"", strings.Join(result, ", "), jsql.RESULT)}
	}

	if len(query.Selects) == 0 {
		selectExprs = append(selectExprs, "*")
	}

	return selectExprs
}

/**
* Query: Generates the Oracle SQL SELECT string for the given Query descriptor.
* Uses OFFSET … ROWS FETCH NEXT … ROWS ONLY for pagination (Oracle 12c+).
* Adds ORDER BY (SELECT NULL FROM DUAL) when pagination is requested but no ORDER BY is set.
* @param query *jsql.Query
* @return string, error
**/
func (s *Oracle) Query(query *jsql.Query) (string, error) {
	if len(query.Froms) == 0 {
		return "", fmt.Errorf("query has no FROM source")
	}

	primary := query.Froms[0]

	var sb strings.Builder

	selects := orSelects(query)
	sb.WriteString("SELECT\n")
	sb.WriteString(strings.Join(selects, ",\n"))

	sb.WriteString(fmt.Sprintf("\nFROM %s \"%s\"", orFromRef(primary), primary.As))
	for _, from := range query.Froms[1:] {
		sb.WriteString(fmt.Sprintf(", %s \"%s\"", orFromRef(from), from.As))
	}

	for _, join := range query.Joins {
		sb.WriteString(fmt.Sprintf("\n%s %s \"%s\"", orJoinKeyword(join.Type), orFromRef(join.To), join.To.As))
		if len(join.Condition) > 0 {
			onSQL := orCondsSQL(query.GetField, query.UseSourceField, join.Condition, join.To.As)
			if onSQL != "" {
				sb.WriteString("\n  ON " + onSQL)
			}
		}
	}

	if len(query.Conditions) > 0 {
		whereSQL := orCondsSQL(query.GetField, query.UseSourceField, query.Conditions, primary.As)
		if whereSQL != "" {
			sb.WriteString("\nWHERE " + whereSQL)
		}
	}

	if len(query.GroupsBy) > 0 {
		exprs := make([]string, 0, len(query.GroupsBy))
		for _, name := range query.GroupsBy {
			fld, ok := query.GetField(name)
			if !ok {
				continue
			}
			exprs = append(exprs, orFieldExpr(fld, query.UseSourceField))
		}
		sb.WriteString("\nGROUP BY " + strings.Join(exprs, ", "))
	}

	if len(query.Havings) > 0 {
		havingSQL := orCondsSQL(query.GetField, query.UseSourceField, query.Havings, primary.As)
		if havingSQL != "" {
			sb.WriteString("\nHAVING " + havingSQL)
		}
	}

	hasOrderBy := len(query.OrdersBy) > 0
	if hasOrderBy {
		parts := make([]string, 0, len(query.OrdersBy))
		for _, idx := range query.OrdersBy {
			dir := "ASC"
			if !idx.Sorted {
				dir = "DESC"
			}
			fld, ok := query.GetField(idx.Name)
			if !ok {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s %s", orFieldExpr(fld, query.UseSourceField), dir))
		}
		sb.WriteString("\nORDER BY " + strings.Join(parts, ", "))
	}

	// OFFSET / FETCH — Oracle 12c+ requires ORDER BY before OFFSET ROWS
	if query.Rows > 0 || query.Offset > 0 {
		if !hasOrderBy {
			sb.WriteString("\nORDER BY (SELECT NULL FROM DUAL)")
		}
		sb.WriteString(fmt.Sprintf("\nOFFSET %d ROWS", query.Offset))
		if query.Rows > 0 {
			sb.WriteString(fmt.Sprintf("\nFETCH NEXT %d ROWS ONLY", query.Rows))
		}
	}

	return sb.String(), nil
}
