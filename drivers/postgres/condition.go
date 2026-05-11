package postgres

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* resolveField: Returns the SQL expression for a field, mapping ATTRIB columns
* to their JSONB path expression (_source->>'field') and COLUMN columns to
* their qualified name (alias.field).
* @param field string
* @param model *jsql.Model
* @param alias string
* @return string
**/
func resolveField(field string, model *jsql.Model, alias string) string {
	col := model.FindColumn(field)
	if col == nil || col.TypeColumn == jsql.COLUMN {
		if alias != "" && !strings.Contains(field, ".") {
			return fmt.Sprintf("%s.%s", alias, field)
		}
		return field
	}

	if col.TypeColumn == jsql.ATTRIB {
		src := model.SourceField
		if alias != "" {
			src = fmt.Sprintf("%s.%s", alias, src)
		}
		cast := pgAttribCast(col.TypeData)
		if cast != "" {
			return fmt.Sprintf("(%s->>'%s')::%s", src, field, cast)
		}
		return fmt.Sprintf("%s->>'%s'", src, field)
	}

	if alias != "" {
		return fmt.Sprintf("%s.%s", alias, field)
	}
	return field
}

/**
* pgAttribCast: Returns the PostgreSQL cast suffix for a JSONB text extraction
* when the target TypeData requires a non-text comparison.
* Returns empty string for text types (no cast needed).
* @param tp jsql.TypeData
* @return string
**/
func pgAttribCast(tp jsql.TypeData) string {
	switch tp {
	case jsql.INT:
		return "BIGINT"
	case jsql.FLOAT:
		return "DOUBLE PRECISION"
	case jsql.BOOLEAN:
		return "BOOLEAN"
	case jsql.DATETIME:
		return "TIMESTAMPTZ"
	default:
		return ""
	}
}

/**
* buildInList: Formats a value slice as a comma-separated SQL IN list.
* @param val any
* @return string
**/
func buildInList(val any) string {
	switch v := val.(type) {
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = fmt.Sprintf("%v", jsql.Quoted(item))
		}
		return strings.Join(parts, ", ")
	case []string:
		parts := make([]string, len(v))
		for i, s := range v {
			parts[i] = fmt.Sprintf("'%s'", s)
		}
		return strings.Join(parts, ", ")
	case []int:
		parts := make([]string, len(v))
		for i, n := range v {
			parts[i] = fmt.Sprintf("%d", n)
		}
		return strings.Join(parts, ", ")
	case []int64:
		parts := make([]string, len(v))
		for i, n := range v {
			parts[i] = fmt.Sprintf("%d", n)
		}
		return strings.Join(parts, ", ")
	case []float64:
		parts := make([]string, len(v))
		for i, n := range v {
			parts[i] = fmt.Sprintf("%v", n)
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", jsql.Quoted(val))
	}
}

/**
* buildCondition: Converts a single et.Condition to a SQL fragment.
* @param cond *et.Condition
* @param model *jsql.Model
* @param alias string
* @return string
**/
func buildCondition(cond *et.Condition, model *jsql.Model, alias string) string {
	field := resolveField(cond.Field, model, alias)

	switch cond.Operator {
	case et.EQ:
		return fmt.Sprintf("%s = %v", field, jsql.Quoted(cond.Value))
	case et.NEG:
		return fmt.Sprintf("%s <> %v", field, jsql.Quoted(cond.Value))
	case et.LESS:
		return fmt.Sprintf("%s < %v", field, jsql.Quoted(cond.Value))
	case et.LESS_EQ:
		return fmt.Sprintf("%s <= %v", field, jsql.Quoted(cond.Value))
	case et.MORE:
		return fmt.Sprintf("%s > %v", field, jsql.Quoted(cond.Value))
	case et.MORE_EQ:
		return fmt.Sprintf("%s >= %v", field, jsql.Quoted(cond.Value))
	case et.LIKE:
		return fmt.Sprintf("%s ILIKE %v", field, jsql.Quoted(cond.Value))
	case et.IN:
		return fmt.Sprintf("%s IN (%s)", field, buildInList(cond.Value))
	case et.NOT_IN:
		return fmt.Sprintf("%s NOT IN (%s)", field, buildInList(cond.Value))
	case et.IS:
		return fmt.Sprintf("%s IS %v", field, jsql.Quoted(cond.Value))
	case et.IS_NOT:
		return fmt.Sprintf("%s IS NOT %v", field, jsql.Quoted(cond.Value))
	case et.NULL:
		return fmt.Sprintf("%s IS NULL", field)
	case et.NOT_NULL:
		return fmt.Sprintf("%s IS NOT NULL", field)
	case et.BETWEEN:
		bv, ok := cond.Value.(et.BetweenValue)
		if !ok {
			return ""
		}
		return fmt.Sprintf("%s BETWEEN %v AND %v", field, jsql.Quoted(bv.Min), jsql.Quoted(bv.Max))
	case et.NOT_BETWEEN:
		bv, ok := cond.Value.(et.BetweenValue)
		if !ok {
			return ""
		}
		return fmt.Sprintf("%s NOT BETWEEN %v AND %v", field, jsql.Quoted(bv.Min), jsql.Quoted(bv.Max))
	default:
		return ""
	}
}

/**
* BuildConditions: Translates a slice of et.Condition to a SQL predicate string
* (without the WHERE keyword). Consecutive conditions are joined with AND/OR
* based on each condition's Connector field.
* @param conds []*et.Condition
* @param model *jsql.Model
* @param alias string
* @return string
**/
func BuildConditions(conds []*et.Condition, model *jsql.Model, alias string) string {
	if len(conds) == 0 {
		return ""
	}

	var sb strings.Builder
	written := 0
	for _, cond := range conds {
		fragment := buildCondition(cond, model, alias)
		if fragment == "" {
			continue
		}
		if written > 0 {
			switch cond.Connector {
			case et.Or:
				sb.WriteString("\n   OR ")
			default:
				sb.WriteString("\n  AND ")
			}
		}
		sb.WriteString(fragment)
		written++
	}

	return sb.String()
}
