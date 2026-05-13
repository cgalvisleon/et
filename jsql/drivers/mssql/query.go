package mssql

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* msFromRef: Returns the [schema].[name] or [name] table reference for FROM/JOIN clauses.
* @param f *jsql.From
* @return string
**/
func msFromRef(f *jsql.From) string {
	if f.Schema != "" {
		return fmt.Sprintf("[%s].[%s]", f.Schema, f.Name)
	}
	return fmt.Sprintf("[%s]", f.Name)
}

/**
* msJoinKeyword: Maps a JoinType to its T-SQL keyword.
* SQL Server does not support FULL JOIN shorthand; emits FULL OUTER JOIN.
* @param tp jsql.JoinType
* @return string
**/
func msJoinKeyword(tp jsql.JoinType) string {
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
* msFieldExpr: Resolves a logical field name to a T-SQL expression.
* COLUMN fields are [bracket]-quoted; ATTRIB fields use JSON_VALUE with optional CAST.
* @param field *jsql.Field, useSource bool
* @return string
**/
func msFieldExpr(field *jsql.Field, useSource bool) string {
	if field == nil || field.From == nil {
		return ""
	}
	alias := field.From.As
	if field.TypeColumn == jsql.COLUMN {
		if alias == "" {
			return fmt.Sprintf("[%s]", field.Name)
		}
		return fmt.Sprintf("[%s].[%s]", alias, field.Name)
	}
	if field.TypeColumn == jsql.ATTRIB {
		if useSource {
			sourceField := jsql.SOURCE
			if field.From.Model != nil && field.From.Model.SourceField != "" {
				sourceField = field.From.Model.SourceField
			}
			col := fmt.Sprintf("[%s]", sourceField)
			if alias != "" {
				col = fmt.Sprintf("[%s].[%s]", alias, sourceField)
			}
			return msJsonValue(col, field.Name, field.TypeData)
		}
		if alias == "" {
			return fmt.Sprintf("[%s]", field.Name)
		}
		return fmt.Sprintf("[%s].[%s]", alias, field.Name)
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
* msInValues: Formats a Go slice as a comma-separated SQL IN-list using reflection.
* @param val any
* @return string
**/
func msInValues(val any) string {
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
* msCondExpr: Renders a single Condition as a T-SQL fragment.
* ATTRIB fields resolve via JSON_VALUE; LIKE replaces ILIKE.
* @param getField func(string) (*jsql.Field, bool), useSourceField bool, cond *et.Condition, alias string
* @return string
**/
func msCondExpr(getField func(string) (*jsql.Field, bool), useSourceField bool, cond *et.Condition, alias string) string {
	var fieldExpr string
	if fld, ok := getField(cond.Field); ok {
		fieldExpr = msFieldExpr(fld, useSourceField)
	}
	if fieldExpr == "" {
		f := cond.Field
		if strings.Contains(f, "->") {
			segs := strings.SplitN(f, "->", 2)
			col := segs[0]
			if alias != "" && !strings.Contains(col, ".") {
				col = fmt.Sprintf("[%s].[%s]", alias, col)
			} else {
				col = fmt.Sprintf("[%s]", col)
			}
			jsonPath := "'$." + strings.ReplaceAll(segs[1], "->", ".") + "'"
			fieldExpr = fmt.Sprintf("JSON_VALUE(%s, %s)", col, jsonPath)
		} else {
			if alias != "" && !strings.Contains(f, ".") {
				f = fmt.Sprintf("[%s].[%s]", alias, f)
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
		return fmt.Sprintf("%s IN (%s)", fieldExpr, msInValues(cond.Value))
	case et.NOT_IN:
		return fmt.Sprintf("%s NOT IN (%s)", fieldExpr, msInValues(cond.Value))
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
* msCondsSQL: Renders a Condition slice as a T-SQL clause body joined by AND/OR connectors.
* @param getField func(string) (*jsql.Field, bool), useSourceField bool, conds []*et.Condition, alias string
* @return string
**/
func msCondsSQL(getField func(string) (*jsql.Field, bool), useSourceField bool, conds []*et.Condition, alias string) string {
	var parts []string
	first := true
	for _, cond := range conds {
		expr := msCondExpr(getField, useSourceField, cond, alias)
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
* msSelectExpr: Resolves an explicit field name from query.Selects into a T-SQL expression.
* ATTRIB columns use JSON_VALUE with appropriate CAST; identifiers are [bracket]-quoted.
* @param query *jsql.Query, field string
* @return string, bool
**/
func msSelectExpr(query *jsql.Query, field string) (string, bool) {
	if fld, ok := findField(query, field); ok {
		alias := fld.From.As
		if fld.TypeColumn == jsql.COLUMN {
			if query.UseSourceField {
				return fmt.Sprintf("'%s', [%s].[%s]", fld.As, alias, fld.Name), true
			}
			return fmt.Sprintf("[%s].[%s] AS [%s]", alias, fld.Name, fld.As), true
		}
		if fld.TypeColumn == jsql.ATTRIB {
			sourceField := jsql.SOURCE
			col := fmt.Sprintf("[%s]", sourceField)
			if alias != "" {
				col = fmt.Sprintf("[%s].[%s]", alias, sourceField)
			}
			expr := msJsonValue(col, fld.Name, fld.TypeData)
			if query.UseSourceField {
				return fmt.Sprintf("'%s', %s", fld.As, expr), true
			}
			return fmt.Sprintf("%s AS [%s]", expr, fld.As), true
		}
	}
	return field, false
}

/**
* msSelects: Generates the SQL SELECT expression list for the given Query descriptor.
* In UseSourceField mode uses JSON_MODIFY chaining instead of PostgreSQL jsonb operators.
* @param query *jsql.Query
* @return []string
**/
func msSelects(query *jsql.Query) []string {
	var selectExprs []string

	if len(query.Selects) > 0 {
		for _, field := range query.Selects {
			if slices.Contains(query.Hiddens, field) {
				continue
			}
			selectExpr, ok := msSelectExpr(query, field)
			if !ok {
				continue
			}
			selectExprs = append(selectExprs, selectExpr)
		}
	}

	if query.UseSourceField {
		if len(query.Selects) > 0 {
			// Build JSON_OBJECT-equivalent using (SELECT ... FOR JSON PATH, WITHOUT_ARRAY_WRAPPER)
			selectExprs = append([]string{}, fmt.Sprintf(
				"(SELECT %s FOR JSON PATH, WITHOUT_ARRAY_WRAPPER) AS [%s]",
				strings.Join(selectExprs, ", "), jsql.RESULT))
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
					if col.TypeColumn != jsql.COLUMN || col.Name == jsql.SOURCE {
						continue
					}
					fld, exists := query.GetField(col.Name)
					if !exists {
						continue
					}
					columnExprs = append(columnExprs, fmt.Sprintf("[%s].[%s] AS [%s]", fld.From.As, col.Name, fld.As))
				}
				// Merge _source with regular columns using JSON_MODIFY chain
				expr := fmt.Sprintf("[%s].[%s]", from.As, jsql.SOURCE)
				for _, ce := range columnExprs {
					// parse "alias.col AS alias" pattern to extract col name and value
					parts := strings.Split(ce, " AS ")
					if len(parts) == 2 {
						colName := strings.Trim(parts[1], "[]")
						colRef := strings.TrimSpace(parts[0])
						expr = fmt.Sprintf("JSON_MODIFY(%s, '$.%s', %s)", expr, colName, colRef)
					}
				}
				selectExprs = append(selectExprs, expr)
			}
			selectExprs = append([]string{}, fmt.Sprintf("%s AS [%s]", strings.Join(selectExprs, ", "), jsql.RESULT))
		}
	} else {
		if len(query.Selects) == 0 {
			selectExprs = append(selectExprs, "*")
		}
	}

	return selectExprs
}

/**
* Query: Generates the T-SQL SELECT string for the given Query descriptor.
* Uses OFFSET … ROWS FETCH NEXT … ROWS ONLY for pagination (requires SQL Server 2012+).
* @param query *jsql.Query
* @return string, error
**/
func (s *MSSQL) Query(query *jsql.Query) (string, error) {
	if len(query.Froms) == 0 {
		return "", fmt.Errorf("query has no FROM source")
	}

	primary := query.Froms[0]

	var sb strings.Builder

	selects := msSelects(query)
	sb.WriteString("SELECT\n")
	sb.WriteString(strings.Join(selects, ",\n"))

	sb.WriteString(fmt.Sprintf("\nFROM %s AS [%s]", msFromRef(primary), primary.As))
	for _, from := range query.Froms[1:] {
		sb.WriteString(fmt.Sprintf(", %s AS [%s]", msFromRef(from), from.As))
	}

	for _, join := range query.Joins {
		sb.WriteString(fmt.Sprintf("\n%s %s AS [%s]", msJoinKeyword(join.Type), msFromRef(join.To), join.To.As))
		if len(join.Condition) > 0 {
			onSQL := msCondsSQL(query.GetField, query.UseSourceField, join.Condition, join.To.As)
			if onSQL != "" {
				sb.WriteString("\n  ON " + onSQL)
			}
		}
	}

	if len(query.Conditions) > 0 {
		whereSQL := msCondsSQL(query.GetField, query.UseSourceField, query.Conditions, primary.As)
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
			exprs = append(exprs, msFieldExpr(fld, query.UseSourceField))
		}
		sb.WriteString("\nGROUP BY " + strings.Join(exprs, ", "))
	}

	if len(query.Havings) > 0 {
		havingSQL := msCondsSQL(query.GetField, query.UseSourceField, query.Havings, primary.As)
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
			parts = append(parts, fmt.Sprintf("%s %s", msFieldExpr(fld, query.UseSourceField), dir))
		}
		sb.WriteString("\nORDER BY " + strings.Join(parts, ", "))
	}

	// OFFSET / FETCH — T-SQL requires ORDER BY before OFFSET ROWS
	if query.Rows > 0 || query.Offset > 0 {
		if !hasOrderBy {
			sb.WriteString("\nORDER BY (SELECT NULL)")
		}
		sb.WriteString(fmt.Sprintf("\nOFFSET %d ROWS", query.Offset))
		if query.Rows > 0 {
			sb.WriteString(fmt.Sprintf("\nFETCH NEXT %d ROWS ONLY", query.Rows))
		}
	}

	sb.WriteString(";")
	return sb.String(), nil
}
