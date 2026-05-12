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
* pgFieldExpr: Resolves a logical field name to a SQL expression, prefixing unqualified
* names with alias and expanding '->' JSONB paths.
* @param alias string
* @param field string
* @return string
**/
func pgFieldExpr(alias, field string) string {
	if strings.Contains(field, ".") {
		return pgJsonbPath(field)
	}
	if alias != "" {
		field = fmt.Sprintf("%s.%s", alias, field)
	}
	return pgJsonbPath(field)
}

/**
* pgAttribExpr: Builds a _source JSONB extraction expression for an ATTRIB column
* with an optional type cast based on the column's TypeData.
* @param alias string
* @param sourceField string
* @param field string
* @param tp jsql.TypeData
* @return string
**/
func pgAttribExpr(alias, sourceField, field string, tp jsql.TypeData) string {
	path := fmt.Sprintf("%s.%s->>'%s'", alias, sourceField, field)
	switch tp {
	case jsql.INT:
		return fmt.Sprintf("(%s)::bigint AS %s", path, field)
	case jsql.FLOAT:
		return fmt.Sprintf("(%s)::double precision AS %s", path, field)
	case jsql.BOOLEAN:
		return fmt.Sprintf("(%s)::boolean AS %s", path, field)
	case jsql.DATETIME:
		return fmt.Sprintf("(%s)::timestamptz AS %s", path, field)
	default:
		return fmt.Sprintf("%s AS %s", path, field)
	}
}

/**
* findAttrib: Returns the Column for field if it is an explicitly-defined ATTRIB in the model.
* @param model *jsql.Model
* @param field string
* @return *jsql.Column, bool
**/
func findAttrib(model *jsql.Model, field string) (*jsql.Column, bool) {
	if model == nil {
		return nil, false
	}
	for _, col := range model.Columns {
		if col.Name == field && col.TypeColumn == jsql.ATTRIB {
			return col, true
		}
		if col.Name == field {
			return col, true
		}
	}
	return nil, false
}

/**
* autoSelectFrom: Builds the SELECT expression list from a From's Model columns.
* Skips the SourceField JSONB blob itself; expands ATTRIB columns inline.
* Excludes any field listed in hiddens.
* @param from *jsql.From
* @param hiddens []string
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
		exprs = []string{}
		exprs = append(exprs, fmt.Sprintf("jsonb_build_object(to_jsonb(%s) - ARRAY[%s]) AS result", from.As, strs.JoinQuoted(columns, ", ")))
	}
	return exprs
}

/**
* resolveSelectField: Resolves an explicit field name from query.Selects into a SQL expression.
* Detects ATTRIB columns and emits the appropriate _source extraction.
* @param from *jsql.From
* @param field string
* @return string
**/
func resolveSelectField(from *jsql.From, field string) string {
	model := from.Model
	useSourceField := model != nil && model.SourceField != ""
	if useSourceField {
		if col, ok := findAttrib(model, field); ok {
			return pgAttribExpr(from.As, model.SourceField, col.Name, col.TypeData)
		}
	}
	return pgFieldExpr(from.As, field)
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
* @param cond *et.Condition
* @param alias string
* @return string
**/
func pgCondExpr(cond *et.Condition, alias string) string {
	fieldExpr := pgFieldExpr(alias, cond.Field)
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
* @param conds []*et.Condition
* @param alias string
* @return string
**/
func pgCondsSQL(conds []*et.Condition, alias string) string {
	var parts []string
	for i, cond := range conds {
		expr := pgCondExpr(cond, alias)
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
			selectExprs = append(selectExprs, resolveSelectField(primary, field))
		}
	} else {
		for _, from := range query.Froms {
			selectExprs = append(selectExprs, autoSelectFrom(from, query.Hiddens)...)
		}
	}

	sb.WriteString("SELECT\n")
	if len(selectExprs) == 0 {
		sb.WriteString("  *")
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
			onSQL := pgCondsSQL(join.Condition, join.To.As)
			if onSQL != "" {
				sb.WriteString("\n  ON " + onSQL)
			}
		}
	}

	// WHERE
	if len(query.Conditions) > 0 {
		whereSQL := pgCondsSQL(query.Conditions, primary.As)
		if whereSQL != "" {
			sb.WriteString("\nWHERE " + whereSQL)
		}
	}

	// GROUP BY
	if len(query.GroupsBy) > 0 {
		exprs := make([]string, 0, len(query.GroupsBy))
		for _, f := range query.GroupsBy {
			exprs = append(exprs, pgFieldExpr(primary.As, f))
		}
		sb.WriteString("\nGROUP BY " + strings.Join(exprs, ", "))
	}

	// HAVING
	if len(query.Havings) > 0 {
		havingSQL := pgCondsSQL(query.Havings, primary.As)
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
			parts = append(parts, fmt.Sprintf("%s %s", pgFieldExpr(primary.As, idx.Name), dir))
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
