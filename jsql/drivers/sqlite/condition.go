package sqlite

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* resolveField: Translates a field name (possibly a nested JSON path using "->" as
* separator) to its SQLite SQL expression.
*
* Rules:
*   - "field"       → alias.field                              (COLUMN)
*   - "field"       → json_extract(alias._source, '$.field')   (ATTRIB)
*   - "field->a->b"  → json_extract(alias.field, '$.a.b')        (COLUMN with JSON path)
*   - "field->a->b"  → json_extract(alias._source, '$.field.a.b') (ATTRIB with JSON path)
*
* A CAST(... AS type) is applied only when the root is a typed ATTRIB with no
* sub-path and its TypeData is numeric, boolean or datetime.
*
* @param field string, model *jsql.Model, alias string
* @return string
**/
func resolveField(field string, model *jsql.Model, alias string) string {
	parts := strings.Split(field, "->")
	root := parts[0]
	path := parts[1:]

	col, ok := model.GetColumn(root)
	isAttrib := ok && col.TypeColumn == jsql.ATTRIB

	if isAttrib {
		src := model.SourceField
		if alias != "" {
			src = fmt.Sprintf("%s.%s", alias, src)
		}
		jsonPath := "$." + strings.Join(append([]string{root}, path...), ".")
		expr := fmt.Sprintf("json_extract(%s, '%s')", src, jsonPath)

		if len(path) == 0 {
			if cast := sqliteAttribCast(col.TypeData); cast != "" {
				return fmt.Sprintf("CAST(%s AS %s)", expr, cast)
			}
		}
		return expr
	}

	var base string
	if alias != "" && !strings.Contains(root, ".") {
		base = fmt.Sprintf("%s.%s", alias, root)
	} else {
		base = root
	}

	if len(path) == 0 {
		return base
	}

	jsonPath := "$." + strings.Join(path, ".")
	return fmt.Sprintf("json_extract(%s, '%s')", base, jsonPath)
}

/**
* sqliteAttribCast: Returns the SQLite CAST type for JSON text extraction when the
* ATTRIB TypeData requires a non-text comparison. Returns empty string for text
* types (no cast needed).
* @param tp jsql.TypeData
* @return string
**/
func sqliteAttribCast(tp jsql.TypeData) string {
	switch tp {
	case jsql.INT:
		return "INTEGER"
	case jsql.FLOAT:
		return "REAL"
	case jsql.BOOLEAN:
		return "INTEGER"
	case jsql.DATETIME:
		return "TEXT"
	default:
		return ""
	}
}

/**
* BuildSelectField: Returns the SQL expression for a SELECT list entry. For simple
* fields it returns the qualified column name. For nested paths (e.g.
* "data->address->city") it returns the JSON path expression followed by
* AS <last_segment> so the result column is named after the leaf key.
* @param field string, model *jsql.Model, alias string
* @return string
**/
func BuildSelectField(field string, model *jsql.Model, alias string) string {
	parts := strings.Split(field, "->")
	expr := resolveField(field, model, alias)

	if len(parts) == 1 {
		return expr
	}

	leaf := parts[len(parts)-1]
	return fmt.Sprintf("%s AS %s", expr, leaf)
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
* buildCondition: Converts a single et.Condition to a SQL predicate fragment.
* Note: SQLite's LIKE is already case-insensitive for ASCII by default, so unlike
* Postgres there is no separate ILIKE operator to translate to.
* @param cond *et.Condition, model *jsql.Model, alias string
* @return string
**/
func buildCondition(cond *et.Condition, model *jsql.Model, alias string) string {
	field := resolveField(cond.Field, model, alias)

	switch cond.Operator {
	case et.EQ:
		return fmt.Sprintf("%s = %v", field, jsql.Quoted(cond.Value.Value))
	case et.NEG:
		return fmt.Sprintf("%s <> %v", field, jsql.Quoted(cond.Value.Value))
	case et.LESS:
		return fmt.Sprintf("%s < %v", field, jsql.Quoted(cond.Value.Value))
	case et.LESS_EQ:
		return fmt.Sprintf("%s <= %v", field, jsql.Quoted(cond.Value.Value))
	case et.MORE:
		return fmt.Sprintf("%s > %v", field, jsql.Quoted(cond.Value.Value))
	case et.MORE_EQ:
		return fmt.Sprintf("%s >= %v", field, jsql.Quoted(cond.Value.Value))
	case et.LIKE:
		return fmt.Sprintf("%s LIKE %v", field, jsql.Quoted(cond.Value.Value))
	case et.IN:
		return fmt.Sprintf("%s IN (%s)", field, buildInList(cond.Value.Value))
	case et.NOT_IN:
		return fmt.Sprintf("%s NOT IN (%s)", field, buildInList(cond.Value.Value))
	case et.IS:
		return fmt.Sprintf("%s IS %v", field, jsql.Quoted(cond.Value.Value))
	case et.IS_NOT:
		return fmt.Sprintf("%s IS NOT %v", field, jsql.Quoted(cond.Value.Value))
	case et.NULL:
		return fmt.Sprintf("%s IS NULL", field)
	case et.NOT_NULL:
		return fmt.Sprintf("%s IS NOT NULL", field)
	case et.BETWEEN:
		bv, ok := cond.Value.Value.(et.BetweenValue)
		if !ok {
			return ""
		}
		return fmt.Sprintf("%s BETWEEN %v AND %v", field, jsql.Quoted(bv.Min), jsql.Quoted(bv.Max))
	case et.NOT_BETWEEN:
		bv, ok := cond.Value.Value.(et.BetweenValue)
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
* according to each condition's Connector field.
* @param conds []*et.Condition, model *jsql.Model, alias string
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
