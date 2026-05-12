package postgres

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/strs"
)

/**
* pgFromRef: Returns the qualified table reference (schema.name) for FROM/JOIN clauses.
* @param f *jsql.From
* @return string
**/
func pgFromRef(f *jsql.From) string {
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
* pgSelectExpr: Resolves a logical field name to a SQL expression, prefixing unqualified
* names with alias and expanding '->' JSONB paths.
* @param alias string
* @param field string
* @return string
**/
func pgSelectExpr(field *jsql.Field, useSource bool) string {
	if field == nil {
		return ""
	}
	if field.From == nil {
		return ""
	}
	if field.TypeColumn == jsql.ATTRIB {
		return pgAttribExpr(field.From.As, jsql.SOURCE, field.Name, field.TypeData)
	} else if field.TypeColumn == jsql.COLUMN {
		if useSource {
			return fmt.Sprintf("'%s', %s.%s", field.As, field.From.As, field.Name)
		} else {
			return fmt.Sprintf("%s.%s AS %s", field.From.As, field.Name, field.As)
		}
	}
	return ""
}

/**
* pgFieldExpr: Resolves a logical field name to a SQL expression, prefixing unqualified
* names with alias and expanding '->' JSONB paths.
* @param alias string
* @param field string
* @return string
**/
func pgFieldExpr(field *jsql.Field, useSource bool) string {
	if field == nil {
		return ""
	}
	if field.From == nil {
		return ""
	}
	if field.TypeColumn == jsql.COLUMN {
		return fmt.Sprintf("%s.%s", field.From.As, field.Name)
	} else if field.TypeColumn == jsql.ATTRIB {
		if useSource {
			fld := pgJsonbPath(field.Name)
			path := fmt.Sprintf("%s.%s->>'%s'", field.From.As, jsql.SOURCE, fld)
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
			return fmt.Sprintf("%s.%s", field.From.As, field.Name)
		}
	}
	return ""
}

/**
* pgAttribExpr: Builds a _source JSONB extraction expression for an ATTRIB column
* with an optional type cast based on the column's TypeData.
* @param alias string, sourceField string, field string, tp jsql.TypeData
* @return string
**/
func pgAttribExpr(alias, sourceField, field string, tp jsql.TypeData) string {
	field = pgJsonbPath(field)
	path := fmt.Sprintf("%s.%s->>'%s'", alias, sourceField, field)
	switch tp {
	case jsql.INT:
		return fmt.Sprintf("'%s', (%s)::bigint", field, path)
	case jsql.FLOAT:
		return fmt.Sprintf("'%s', (%s)::double precision", field, path)
	case jsql.BOOLEAN:
		return fmt.Sprintf("'%s', (%s)::boolean", field, path)
	case jsql.DATETIME:
		return fmt.Sprintf("'%s', (%s)::timestamptz", field, path)
	default:
		return fmt.Sprintf("'%s', %s", field, path)
	}
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
* autoSelectFrom: Builds the SELECT expression list from a From's Model columns.
* Skips the SourceField JSONB blob itself; expands ATTRIB columns inline.
* Excludes any field listed in hiddens.
* @param from *jsql.From, hiddens []string
* @return []string
**/
func autoSelectFrom(from *jsql.From, hiddens []string) []string {
	model := from.Model
	if model == nil {
		return []string{fmt.Sprintf("%s.*", from.As)}
	}
	useSourceField := model.SourceField != ""
	columns := make([]string, 0)
	exprs := make([]string, 0, len(model.Columns))
	for _, col := range model.Columns {
		if slices.Contains(hiddens, col.Name) {
			continue
		}
		if col.TypeColumn == jsql.COLUMN {
			if col.Name == model.SourceField {
				continue
			}
			exprs = append(exprs, fmt.Sprintf("%s.%s AS %s", from.As, col.Name, col.Name))
			columns = append(columns, col.Name)
		}
	}
	if useSourceField {
		exprs = append([]string{}, fmt.Sprintf("to_jsonb(%s) - ARRAY[%s]", from.As, strs.JoinQuoted(columns, ", ")))
	}
	return exprs
}

/**
* resolveSelectField: Resolves an explicit field name from query.Selects into a SQL expression.
* Detects ATTRIB columns and emits the appropriate _source extraction.
* @param from *jsql.From, field string
* @return string
**/
func resolveSelectField(query *jsql.Query, field string) (string, bool) {
	if fld, ok := findField(query, field); ok {
		return pgSelectExpr(fld, query.UseSourceField), true
	}

	return field, false
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
* @param getField func(string), useSourceField bool, cond *et.Condition, alias string
* @return string
**/
func pgCondExpr(getField func(string) (*jsql.Field, bool), useSourceField bool, cond *et.Condition, alias string) string {
	fld, ok := getField(cond.Field)
	if !ok {
		return ""
	}
	fieldExpr := pgFieldExpr(fld, useSourceField)
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
	for i, cond := range conds {
		expr := pgCondExpr(getField, useSourceField, cond, alias)
		if expr == "" {
			continue
		}
		if i == 0 || cond.Connector == et.NaC {
			parts = append(parts, expr)
		} else if cond.Connector == et.And {
			parts = append(parts, "AND "+expr)
		} else {
			parts = append(parts, "OR "+expr)
		}
	}
	return strings.Join(parts, "\n  ")
}

/**
* Query: Generates the SQL SELECT string for the given Query descriptor.
* @param query *jsql.Query
* @return string, error
**/
func (s *Postgres) Query(query *jsql.Query) (string, error) {
	if len(query.Froms) == 0 {
		return "", fmt.Errorf("query has no FROM source")
	}

	primary := query.Froms[0]

	var sb strings.Builder

	// SELECT
	var selectExprs []string
	if len(query.Selects) > 0 {
		for _, field := range query.Selects {
			if slices.Contains(query.Hiddens, field) {
				continue
			}
			selectExpr, ok := resolveSelectField(query, field)
			if !ok {
				continue
			}
			selectExprs = append(selectExprs, selectExpr)
		}
	} else {
		for _, from := range query.Froms {
			selectExprs = append(selectExprs, autoSelectFrom(from, query.Hiddens)...)
		}
	}

	sb.WriteString("SELECT\n")
	if len(selectExprs) == 0 {
		sb.WriteString("  *")
	} else if query.UseSourceField {
		sb.WriteString(fmt.Sprintf("jsonb_build_object(\n%s\n) AS result", strings.Join(selectExprs, ",\n")))
	} else {
		sb.WriteString("  " + strings.Join(selectExprs, ",\n  "))
	}

	// FROM
	sb.WriteString(fmt.Sprintf("\nFROM %s AS %s", pgFromRef(primary), primary.As))
	for _, from := range query.Froms[1:] {
		sb.WriteString(fmt.Sprintf(", %s AS %s", pgFromRef(from), from.As))
	}

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

	sb.WriteString(";")
	return sb.String(), nil
}
