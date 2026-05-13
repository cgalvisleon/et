package mysql

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* myResolveField: Translates a field name (possibly a nested JSON path using "->" as separator)
* to its MySQL SQL expression.
*
* Rules:
*   - "field"       → alias.field                                             (COLUMN)
*   - "field"       → JSON_UNQUOTE(JSON_EXTRACT(alias._source, '$.field'))    (ATTRIB text)
*   - "field"       → CAST(JSON_EXTRACT(alias._source, '$.field') AS SIGNED)  (ATTRIB int)
*   - "col->a->b"   → JSON_UNQUOTE(JSON_EXTRACT(alias.col, '$.a.b'))          (COLUMN JSON path)
*
* @param field string, model *jsql.Model, alias string
* @return string
**/
func myResolveField(field string, model *jsql.Model, alias string) string {
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
			src = fmt.Sprintf("`%s`.`%s`", alias, src)
		}
		jsonCol = src
		pathSegs = append([]string{root}, path...)
	} else {
		if alias != "" && !strings.Contains(root, ".") {
			jsonCol = fmt.Sprintf("`%s`.`%s`", alias, root)
		} else {
			jsonCol = root
		}
		pathSegs = path
	}

	if len(pathSegs) == 0 {
		return jsonCol
	}

	jsonPath := "'$." + strings.Join(pathSegs, ".") + "'"

	if isAttrib && len(path) == 0 {
		return myJsonExtract(jsonCol, root, col.TypeData)
	}

	return fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(%s, %s))", jsonCol, jsonPath)
}

/**
* MyBuildSelectField: Returns the SQL expression for a SELECT list entry.
* For simple fields returns the qualified column name.
* For nested paths returns the JSON_UNQUOTE(JSON_EXTRACT(...)) expression with AS <last_segment>.
* @param field string, model *jsql.Model, alias string
* @return string
**/
func MyBuildSelectField(field string, model *jsql.Model, alias string) string {
	parts := strings.Split(field, "->")
	expr := myResolveField(field, model, alias)

	if len(parts) == 1 {
		return expr
	}

	leaf := parts[len(parts)-1]
	return fmt.Sprintf("%s AS `%s`", expr, leaf)
}

/**
* myBuildInList: Formats a value slice as a comma-separated SQL IN list.
* @param val any
* @return string
**/
func myBuildInList(val any) string {
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
* myBuildCondition: Converts a single et.Condition to a MySQL SQL predicate fragment.
* Uses LIKE (case-insensitive for utf8mb4_unicode_ci collation) instead of ILIKE.
* @param cond *et.Condition, model *jsql.Model, alias string
* @return string
**/
func myBuildCondition(cond *et.Condition, model *jsql.Model, alias string) string {
	field := myResolveField(cond.Field, model, alias)

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
		return fmt.Sprintf("%s IN (%s)", field, myBuildInList(cond.Value))
	case et.NOT_IN:
		return fmt.Sprintf("%s NOT IN (%s)", field, myBuildInList(cond.Value))
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
* MyBuildConditions: Translates a slice of et.Condition to a SQL predicate string
* (without the WHERE keyword). Conditions are joined with AND/OR per Connector field.
* @param conds []*et.Condition, model *jsql.Model, alias string
* @return string
**/
func MyBuildConditions(conds []*et.Condition, model *jsql.Model, alias string) string {
	if len(conds) == 0 {
		return ""
	}

	var sb strings.Builder
	written := 0
	for _, cond := range conds {
		fragment := myBuildCondition(cond, model, alias)
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
