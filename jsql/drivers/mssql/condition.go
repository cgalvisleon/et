package mssql

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* msResolveField: Translates a field name (possibly a nested JSON path using "->" as separator)
* to its SQL Server SQL expression.
*
* Rules:
*   - "field"       → [alias].[field]                                          (COLUMN)
*   - "field"       → JSON_VALUE([alias].[_source], '$.field')                 (ATTRIB text)
*   - "field"       → CAST(JSON_VALUE([alias].[_source], '$.field') AS BIGINT) (ATTRIB int)
*   - "col->a->b"   → JSON_VALUE([alias].[col], '$.a.b')                       (COLUMN JSON path)
*
* @param field string, model *jsql.Model, alias string
* @return string
**/
func msResolveField(field string, model *jsql.Model, alias string) string {
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
			src = fmt.Sprintf("[%s].[%s]", alias, src)
		} else {
			src = fmt.Sprintf("[%s]", src)
		}
		jsonCol = src
		pathSegs = append([]string{root}, path...)
	} else {
		if alias != "" && !strings.Contains(root, ".") {
			jsonCol = fmt.Sprintf("[%s].[%s]", alias, root)
		} else {
			jsonCol = fmt.Sprintf("[%s]", root)
		}
		pathSegs = path
	}

	if len(pathSegs) == 0 {
		return jsonCol
	}

	jsonPath := "'$." + strings.Join(pathSegs, ".") + "'"

	if isAttrib && len(path) == 0 {
		return msJsonValue(jsonCol, root, col.TypeData)
	}

	return fmt.Sprintf("JSON_VALUE(%s, %s)", jsonCol, jsonPath)
}

/**
* MsBuildSelectField: Returns the SQL expression for a SELECT list entry.
* For simple fields returns the qualified bracketed column name.
* For nested paths returns JSON_VALUE(...) AS [last_segment].
* @param field string, model *jsql.Model, alias string
* @return string
**/
func MsBuildSelectField(field string, model *jsql.Model, alias string) string {
	parts := strings.Split(field, "->")
	expr := msResolveField(field, model, alias)

	if len(parts) == 1 {
		return expr
	}

	leaf := parts[len(parts)-1]
	return fmt.Sprintf("%s AS [%s]", expr, leaf)
}

/**
* msBuildInList: Formats a value slice as a comma-separated SQL IN list.
* @param val any
* @return string
**/
func msBuildInList(val any) string {
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
			parts[i] = fmt.Sprintf("N'%s'", s)
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
* msBuildCondition: Converts a single et.Condition to a T-SQL predicate fragment.
* Uses LIKE (case behaviour determined by collation) instead of PostgreSQL ILIKE.
* @param cond *et.Condition, model *jsql.Model, alias string
* @return string
**/
func msBuildCondition(cond *et.Condition, model *jsql.Model, alias string) string {
	field := msResolveField(cond.Field, model, alias)

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
		return fmt.Sprintf("%s IN (%s)", field, msBuildInList(cond.Value))
	case et.NOT_IN:
		return fmt.Sprintf("%s NOT IN (%s)", field, msBuildInList(cond.Value))
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
* MsBuildConditions: Translates a slice of et.Condition to a SQL predicate string
* (without the WHERE keyword). Conditions are joined with AND/OR per Connector field.
* @param conds []*et.Condition, model *jsql.Model, alias string
* @return string
**/
func MsBuildConditions(conds []*et.Condition, model *jsql.Model, alias string) string {
	if len(conds) == 0 {
		return ""
	}

	var sb strings.Builder
	written := 0
	for _, cond := range conds {
		fragment := msBuildCondition(cond, model, alias)
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
