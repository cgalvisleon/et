package oracle

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* orResolveField: Translates a field name (possibly a nested JSON path using "->" as separator)
* to its Oracle SQL expression.
*
* Rules:
*   - "field"       → "alias"."field"                                           (COLUMN)
*   - "field"       → JSON_VALUE("alias"."_source", '$.field')                  (ATTRIB text)
*   - "field"       → JSON_VALUE("alias"."_source", '$.field' RETURNING NUMBER) (ATTRIB typed)
*   - "col->a->b"   → JSON_VALUE("alias"."col", '$.a.b')                        (COLUMN JSON path)
*
* @param field string, model *jsql.Model, alias string
* @return string
**/
func orResolveField(field string, model *jsql.Model, alias string) string {
	parts := strings.Split(field, "->")
	root := parts[0]
	path := parts[1:]

	col, ok := model.GetColumn(root)
	isAttrib := ok && col.TypeColumn == jsql.ATTRIB

	var jsonCol string
	var pathSegs []string

	if isAttrib {
		src := model.SourceField
		if alias != "" {
			jsonCol = fmt.Sprintf(`"%s"."%s"`, alias, src)
		} else {
			jsonCol = fmt.Sprintf(`"%s"`, src)
		}
		pathSegs = append([]string{root}, path...)
	} else {
		if alias != "" && !strings.Contains(root, ".") {
			jsonCol = fmt.Sprintf(`"%s"."%s"`, alias, root)
		} else {
			jsonCol = fmt.Sprintf(`"%s"`, root)
		}
		pathSegs = path
	}

	if len(pathSegs) == 0 {
		return jsonCol
	}

	jsonPath := "'$." + strings.Join(pathSegs, ".") + "'"

	if isAttrib && len(path) == 0 {
		return orJsonValue(jsonCol, root, col.TypeData)
	}

	return fmt.Sprintf("JSON_VALUE(%s, %s)", jsonCol, jsonPath)
}

/**
* OrBuildSelectField: Returns the Oracle SQL expression for a SELECT list entry.
* For simple fields returns the qualified double-quoted column name.
* For nested paths returns JSON_VALUE(...) AS "last_segment".
* @param field string, model *jsql.Model, alias string
* @return string
**/
func OrBuildSelectField(field string, model *jsql.Model, alias string) string {
	parts := strings.Split(field, "->")
	expr := orResolveField(field, model, alias)

	if len(parts) == 1 {
		return expr
	}

	leaf := parts[len(parts)-1]
	return fmt.Sprintf(`%s AS "%s"`, expr, leaf)
}

/**
* orBuildInList: Formats a value slice as a comma-separated SQL IN list.
* @param val any
* @return string
**/
func orBuildInList(val any) string {
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
* orBuildCondition: Converts a single et.Condition to an Oracle SQL predicate fragment.
* Uses LIKE (case behaviour follows NLS_COMP/NLS_SORT settings) instead of PostgreSQL ILIKE.
* @param cond *et.Condition, model *jsql.Model, alias string
* @return string
**/
func orBuildCondition(cond *et.Condition, model *jsql.Model, alias string) string {
	field := orResolveField(cond.Field, model, alias)

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
		return fmt.Sprintf("%s LIKE %v", field, jsql.Quoted(cond.Value))
	case et.IN:
		return fmt.Sprintf("%s IN (%s)", field, orBuildInList(cond.Value))
	case et.NOT_IN:
		return fmt.Sprintf("%s NOT IN (%s)", field, orBuildInList(cond.Value))
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
* OrBuildConditions: Translates a slice of et.Condition to an Oracle SQL predicate string
* (without the WHERE keyword). Conditions are joined with AND/OR per Connector field.
* @param conds []*et.Condition, model *jsql.Model, alias string
* @return string
**/
func OrBuildConditions(conds []*et.Condition, model *jsql.Model, alias string) string {
	if len(conds) == 0 {
		return ""
	}

	var sb strings.Builder
	written := 0
	for _, cond := range conds {
		fragment := orBuildCondition(cond, model, alias)
		if fragment == "" {
			continue
		}
		if written > 0 {
			switch cond.Connector {
			case et.Or:
				sb.WriteString("\n OR ")
			default:
				sb.WriteString("\n AND ")
			}
		}
		sb.WriteString(fragment)
		written++
	}

	return sb.String()
}
