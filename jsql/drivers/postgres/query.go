package postgres

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* pgFromRef: Returns the qualified table reference (schema.name) for FROM/JOIN clauses.
* @param f *jsql.F
* @return string
**/
func pgFromRef(f *jsql.F) string {
	if f.Schema != "" {
		return fmt.Sprintf("%s.%s", f.Schema, f.Name)
	}
	return f.Name
}

/**
* pgJoinKeyword: Maps a JoinType to its SQL keyword.
* @param tp jsql.JoinType
* @return string
**/
func pgJoinKeyword(tp jsql.JoinType) string {
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
* pgJsonbPath: Converts a field reference containing '->' separators into a PostgreSQL
* JSONB path expression: intermediate levels use -> 'key', the last level uses ->>'key'.
* @param field string
* @return string
**/
func pgJsonbPath(field string) string {
	if !strings.Contains(field, "->") {
		return field
	}
	parts := strings.Split(field, "->")
	var sb strings.Builder
	sb.WriteString(parts[0])
	for i, p := range parts[1:] {
		if i == len(parts)-2 {
			sb.WriteString(fmt.Sprintf("->>'%s'", p))
		} else {
			sb.WriteString(fmt.Sprintf("->'%s'", p))
		}
	}
	return sb.String()
}

/**
* pgFieldExpr: Resolves a logical field name to a SQL expression, prefixing unqualified
* names with alias and expanding '->' JSONB paths.
* @param field *jsql.Field, useSource bool
* @return string
**/
func pgFieldExpr(field *jsql.Field, useSource bool) string {
	if field == nil {
		return ""
	}
	if field.From == nil {
		return ""
	}
	alias := field.From.As
	if field.TypeColumn == jsql.COLUMN {
		if alias == "" {
			return field.Name
		}
		return fmt.Sprintf("%s.%s", alias, field.Name)
	} else if field.TypeColumn == jsql.ATTRIB {
		if useSource {
			sourceField := jsql.SOURCE
			if field.From.Model != nil && field.From.Model.SourceField != "" {
				sourceField = field.From.Model.SourceField
			}
			fullPath := pgJsonbPath(sourceField + "->" + field.Name)
			path := fullPath
			if alias != "" {
				path = alias + "." + fullPath
			}
			switch field.TypeData {
			case jsql.INT:
				return fmt.Sprintf("(%s)::bigint", path)
			case jsql.FLOAT:
				return fmt.Sprintf("(%s)::double precision", path)
			case jsql.BOOLEAN:
				return fmt.Sprintf("(%s)::boolean", path)
			case jsql.DATETIME:
				return fmt.Sprintf("(%s)::timestamptz", path)
			default:
				return path
			}
		} else {
			if alias == "" {
				return field.Name
			}
			return fmt.Sprintf("%s.%s", alias, field.Name)
		}
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
* pgInValues: Formats a Go slice as a comma-separated SQL IN-list.
* @param val any
* @return string
**/
func pgInValues(val any) string {
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
* pgCondExpr: Renders a single Condition as a SQL fragment using alias to qualify the field.
* @param getField func(string) (*jsql.Field, bool), useSourceField bool, cond *et.Condition, alias string
* @return string
**/
func pgCondExpr(getField func(string) (*jsql.Field, bool), useSourceField bool, cond *et.Condition, alias string) string {
	var fieldExpr string
	if fld, ok := getField(cond.Field); ok {
		fieldExpr = pgFieldExpr(fld, useSourceField)
	}
	if fieldExpr == "" {
		f := pgJsonbPath(cond.Field)
		if alias != "" && !strings.Contains(f, ".") {
			f = fmt.Sprintf("%s.%s", alias, f)
		}
		fieldExpr = f
	}
	switch cond.Operator {
	case et.NULL:
		return fmt.Sprintf("%s IS NULL", fieldExpr)
	case et.NOT_NULL:
		return fmt.Sprintf("%s IS NOT NULL", fieldExpr)
	case et.IN:
		return fmt.Sprintf("%s IN (%s)", fieldExpr, pgInValues(cond.Value))
	case et.NOT_IN:
		return fmt.Sprintf("%s NOT IN (%s)", fieldExpr, pgInValues(cond.Value))
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
		return fmt.Sprintf("%s ILIKE %v", fieldExpr, jsql.Quoted(cond.Value))
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
* pgCondsSQL: Renders a Condition slice as a SQL clause body joined by AND/OR connectors.
* @param getField func(string) (*jsql.Field, bool), useSourceField bool, conds []*et.Condition, alias string
* @return string
**/
func pgCondsSQL(getField func(string) (*jsql.Field, bool), useSourceField bool, conds []*et.Condition, alias string) string {
	var parts []string
	first := true
	for _, cond := range conds {
		expr := pgCondExpr(getField, useSourceField, cond, alias)
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
* pgSelectExpr: Resolves an explicit field name from query.Selects into a SQL expression.
* Detects ATTRIB columns and emits the appropriate _source extraction.
* @param query *jsql.Query, field string
* @return string, bool
**/
func pgSelectExpr(query *jsql.Query, field string) (string, bool) {
	fld, ok := findField(query, field)
	if !ok {
		return "", false
	}
	alias := fld.From.As
	if fld.TypeColumn == jsql.COLUMN {
		if query.UseSourceField {
			return fmt.Sprintf("'%s', %s.%s", fld.As, alias, fld.Name), true
		} else {
			return fmt.Sprintf("%s.%s AS %s", alias, fld.Name, fld.As), true
		}
	}
	if fld.TypeColumn == jsql.ATTRIB {
		sourceField := jsql.SOURCE
		fullPath := pgJsonbPath(sourceField + "->" + fld.Name)
		path := fullPath
		if alias != "" {
			path = alias + "." + fullPath
		}
		switch fld.TypeData {
		case jsql.INT:
			return fmt.Sprintf("'%s', (%s)::bigint", fld.As, path), true
		case jsql.FLOAT:
			return fmt.Sprintf("'%s', (%s)::double precision", fld.As, path), true
		case jsql.BOOLEAN:
			return fmt.Sprintf("'%s', (%s)::boolean", fld.As, path), true
		case jsql.DATETIME:
			return fmt.Sprintf("'%s', (%s)::timestamptz", fld.As, path), true
		default:
			return fmt.Sprintf("'%s', %s", fld.As, path), true
		}
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
		query.Details[fld.Name] = &jsql.QueryDetail{
			To:     detail.To,
			Keys:   detail.Keys,
			Select: detail.Select,
			Page:   fld.Page,
			Rows:   detail.Rows,
		}
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
		query.Rollups[fld.Name] = &jsql.QueryDetail{
			To:     rollup.To,
			Keys:   rollup.Keys,
			Select: rollup.Select,
			Page:   fld.Page,
			Rows:   rollup.Rows,
		}
	}
	if fld.TypeColumn == jsql.CALCFUNC {
		if fld.From == nil {
			return "", false
		}
		if fld.From.Model == nil {
			return "", false
		}
		calc, ok := fld.From.Model.GetCalcFunc(fld.Name)
		if ok {
			query.CalcFuns[fld.Name] = calc
		}
	}
	if fld.TypeColumn == jsql.CALC {
		if fld.From == nil {
			return "", false
		}
		if fld.From.Model == nil {
			return "", false
		}
		calc, ok := fld.From.Model.Calcs[fld.Name]
		if ok {
			query.Calcs[fld.Name] = calc
		}
	}

	return "", false
}

/**
* pgSelects: Generates the SQL SELECT string for the given Query descriptor.
* @param query *jsql.Query
* @return string
**/
func pgSelects(query *jsql.Query) []string {
	var selectExprs []string
	if len(query.Selects) > 0 {
		for _, field := range query.Selects {
			if slices.Contains(query.Hiddens, field) {
				continue
			}
			if field == jsql.SOURCE {
				continue
			}
			selectExpr, ok := pgSelectExpr(query, field)
			if !ok {
				continue
			}
			selectExprs = append(selectExprs, selectExpr)
		}
		selectExprs = append([]string{}, fmt.Sprintf("jsonb_build_object(\n%s\n)", strings.Join(selectExprs, ",\n")))
	} else {
		for _, from := range query.Froms {
			model := from.Model
			var columnExprs []string
			for _, col := range model.Columns {
				if slices.Contains(query.Hiddens, col.Name) {
					continue
				}
				if slices.Contains(model.Hiddens, col.Name) {
					continue
				}
				if col.TypeColumn != jsql.COLUMN && col.Name == jsql.SOURCE {
					continue
				}
				columnExpr, ok := pgSelectExpr(query, col.Name)
				if !ok {
					continue
				}
				columnExprs = append(columnExprs, columnExpr)
			}
			selectExprs = append(selectExprs, fmt.Sprintf("%s.%s ||\njsonb_build_object(\n%s)", from.As, jsql.SOURCE, strings.Join(columnExprs, ",\n")))
		}
	}

	return append([]string{}, fmt.Sprintf("%s AS %s", strings.Join(selectExprs, ",\n"), jsql.RESULT))
}

/**
* pgFrom: Generates the SQL FROM string for the given Query descriptor.
* @param query *jsql.Query
* @return strings.Builder
**/
func pgFrom(query *jsql.Query) []string {
	result := []string{}
	primary := query.Froms[0]
	ref := pgFromRef(primary)
	if ref == primary.As {
		result = append(result, fmt.Sprintf("\nFROM %s", ref))
	} else {
		result = append(result, fmt.Sprintf("\nFROM %s AS %s", ref, primary.As))
	}
	for _, from := range query.Froms[1:] {
		ref := pgFromRef(from)
		if ref == from.As {
			result = append(result, fmt.Sprintf(",\n%s", ref))
		} else {
			result = append(result, fmt.Sprintf(",\n%s AS %s", ref, from.As))
		}
	}

	return result
}

/**
* Query: Generates the SQL SELECT string for the given Query descriptor.
* @param query *jsql.Query
* @return string, error
*
 */
func (s *Postgres) Query(query *jsql.Query) (string, error) {
	if len(query.Froms) == 0 {
		return "", fmt.Errorf("query has no FROM source")
	}

	primary := query.Froms[0]

	var sb strings.Builder
	if query.IsExists {
		// EXISTS
		sb.WriteString("SELECT 1")
	} else {
		// SELECT
		selects := pgSelects(query)
		sb.WriteString("SELECT\n")
		sb.WriteString(strings.Join(selects, ",\n"))
	}

	// FROM
	from := pgFrom(query)
	sb.WriteString(strings.Join(from, ",\n"))

	// JOINs
	for _, join := range query.Joins {
		sb.WriteString(fmt.Sprintf("\n%s %s AS %s", pgJoinKeyword(join.Type), pgFromRef(join.To), join.To.As))
		if len(join.Condition) > 0 {
			onSQL := pgCondsSQL(query.GetField, query.UseSourceField, join.Condition, join.To.As)
			if onSQL != "" {
				sb.WriteString("\n  ON " + onSQL)
			}
		}
	}

	// WHERE
	if len(query.Conditions) > 0 {
		whereSQL := pgCondsSQL(query.GetField, query.UseSourceField, query.Conditions, primary.As)
		if whereSQL != "" {
			sb.WriteString("\nWHERE " + whereSQL)
		}
	}

	// GROUP BY
	if len(query.GroupsBy) > 0 {
		exprs := make([]string, 0, len(query.GroupsBy))
		for _, name := range query.GroupsBy {
			fld, ok := query.GetField(name)
			if !ok {
				continue
			}
			exprs = append(exprs, pgFieldExpr(fld, query.UseSourceField))
		}
		sb.WriteString("\nGROUP BY " + strings.Join(exprs, ", "))
	}

	// HAVING
	if len(query.Havings) > 0 {
		havingSQL := pgCondsSQL(query.GetField, query.UseSourceField, query.Havings, primary.As)
		if havingSQL != "" {
			sb.WriteString("\nHAVING " + havingSQL)
		}
	}

	// ORDER BY
	if len(query.OrdersBy) > 0 {
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
			parts = append(parts, fmt.Sprintf("%s %s", pgFieldExpr(fld, query.UseSourceField), dir))
		}
		sb.WriteString("\nORDER BY " + strings.Join(parts, ", "))
	}

	// LIMIT / OFFSET
	if query.Rows > 0 {
		sb.WriteString(fmt.Sprintf("\nLIMIT %d", query.Rows))
	}
	if query.Offset > 0 {
		sb.WriteString(fmt.Sprintf("\nOFFSET %d", query.Offset))
	}

	if query.IsExists {
		// For EXISTS queries, we only need a dummy select
		sql := fmt.Sprintf("SELECT EXISTS(%s)", sb.String())
		sb.Reset()
		sb.WriteString(sql)
	}

	sb.WriteString(";")
	return sb.String(), nil
}
