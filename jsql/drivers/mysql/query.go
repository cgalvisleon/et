package mysql

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* myFromRef: Returns the table name for FROM/JOIN clauses.
* MySQL schemas map to databases — table references use only the table name.
* @param f *jsql.From
* @return string
**/
func myFromRef(f *jsql.From) string {
	return f.Name
}

/**
* myJoinKeyword: Maps a JoinType to its MySQL SQL keyword.
* MySQL does not support FULL OUTER JOIN; it falls back to LEFT JOIN.
* @param tp jsql.JoinType
* @return string
**/
func myJoinKeyword(tp jsql.JoinType) string {
	switch tp {
	case jsql.LEFT_JOIN:
		return "LEFT JOIN"
	case jsql.RIGHT_JOIN:
		return "RIGHT JOIN"
	default:
		return "INNER JOIN"
	}
}

/**
* myFieldExpr: Resolves a logical field name to a MySQL SQL expression.
* COLUMN fields are backtick-quoted and prefixed with alias.
* ATTRIB fields use JSON_UNQUOTE(JSON_EXTRACT(...)) with optional CAST.
* @param field *jsql.Field, useSource bool
* @return string
**/
func myFieldExpr(field *jsql.Field, useSource bool) string {
	if field == nil || field.From == nil {
		return ""
	}
	alias := field.From.As
	if field.TypeColumn == jsql.COLUMN {
		if alias == "" {
			return fmt.Sprintf("`%s`", field.Name)
		}
		return fmt.Sprintf("`%s`.`%s`", alias, field.Name)
	}
	if field.TypeColumn == jsql.ATTRIB {
		if useSource {
			sourceField := jsql.SOURCE
			if field.From.Model != nil && field.From.Model.SourceField != "" {
				sourceField = field.From.Model.SourceField
			}
			col := fmt.Sprintf("`%s`", sourceField)
			if alias != "" {
				col = fmt.Sprintf("`%s`.`%s`", alias, sourceField)
			}
			return myJsonExtract(col, field.Name, field.TypeData)
		}
		if alias == "" {
			return fmt.Sprintf("`%s`", field.Name)
		}
		return fmt.Sprintf("`%s`.`%s`", alias, field.Name)
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
* myInValues: Formats a Go slice as a comma-separated SQL IN-list using reflection.
* @param val any
* @return string
**/
func myInValues(val any) string {
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
* myCondExpr: Renders a single Condition as a MySQL SQL fragment.
* ATTRIB fields resolve via JSON_UNQUOTE(JSON_EXTRACT(...)); LIKE replaces ILIKE.
* @param getField func(string) (*jsql.Field, bool), useSourceField bool, cond *et.Condition, alias string
* @return string
**/
func myCondExpr(getField func(string) (*jsql.Field, bool), useSourceField bool, cond *et.Condition, alias string) string {
	var fieldExpr string
	if fld, ok := getField(cond.Field); ok {
		fieldExpr = myFieldExpr(fld, useSourceField)
	}
	if fieldExpr == "" {
		f := cond.Field
		if strings.Contains(f, "->") {
			segs := strings.SplitN(f, "->", 2)
			col := segs[0]
			if alias != "" && !strings.Contains(col, ".") {
				col = fmt.Sprintf("`%s`.`%s`", alias, col)
			}
			jsonPath := "'$." + strings.ReplaceAll(segs[1], "->", ".") + "'"
			fieldExpr = fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(%s, %s))", col, jsonPath)
		} else {
			if alias != "" && !strings.Contains(f, ".") {
				f = fmt.Sprintf("`%s`.`%s`", alias, f)
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
		return fmt.Sprintf("%s IN (%s)", fieldExpr, myInValues(cond.Value))
	case et.NOT_IN:
		return fmt.Sprintf("%s NOT IN (%s)", fieldExpr, myInValues(cond.Value))
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
* myCondsSQL: Renders a Condition slice as a SQL clause body joined by AND/OR connectors.
* @param getField func(string) (*jsql.Field, bool), useSourceField bool, conds []*et.Condition, alias string
* @return string
**/
func myCondsSQL(getField func(string) (*jsql.Field, bool), useSourceField bool, conds []*et.Condition, alias string) string {
	var parts []string
	first := true
	for _, cond := range conds {
		expr := myCondExpr(getField, useSourceField, cond, alias)
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
* mySelectExpr: Resolves an explicit field name from query.Selects into a MySQL SQL expression.
* ATTRIB columns use JSON_UNQUOTE(JSON_EXTRACT(...)) with appropriate CAST.
* @param query *jsql.Query, field string
* @return string, bool
**/
func mySelectExpr(query *jsql.Query, field string) (string, bool) {
	fld, ok := findField(query, field)
	if !ok {
		return "", false
	}
	alias := fld.From.As
	if fld.TypeColumn == jsql.COLUMN {
		if query.UseSourceField {
			return fmt.Sprintf("'%s', `%s`.`%s`", fld.As, alias, fld.Name), true
		}
		return fmt.Sprintf("`%s`.`%s` AS `%s`", alias, fld.Name, fld.As), true
	}
	if fld.TypeColumn == jsql.ATTRIB {
		sourceField := jsql.SOURCE
		col := fmt.Sprintf("`%s`", sourceField)
		if alias != "" {
			col = fmt.Sprintf("`%s`.`%s`", alias, sourceField)
		}
		expr := myJsonExtract(col, fld.Name, fld.TypeData)
		if query.UseSourceField {
			return fmt.Sprintf("'%s', %s", fld.As, expr), true
		}
		return fmt.Sprintf("%s AS `%s`", expr, fld.As), true
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
* mySelects: Generates the SQL SELECT expression list for the given Query descriptor.
* In UseSourceField mode uses JSON_OBJECT / JSON_MERGE_PATCH instead of PostgreSQL jsonb equivalents.
* JSON_MERGE_PATCH requires MySQL 8.0+.
* @param query *jsql.Query
* @return []string
**/
func mySelects(query *jsql.Query) []string {
	var selectExprs []string

	if len(query.Selects) > 0 {
		for _, field := range query.Selects {
			if slices.Contains(query.Hiddens, field) {
				continue
			}
			selectExpr, ok := mySelectExpr(query, field)
			if !ok {
				continue
			}
			selectExprs = append(selectExprs, selectExpr)
		}
	}

	if query.UseSourceField {
		if len(query.Selects) > 0 {
			selectExprs = append([]string{}, fmt.Sprintf("JSON_OBJECT(\n%s\n) AS `%s`", strings.Join(selectExprs, ",\n"), jsql.RESULT))
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
					columnExprs = append(columnExprs, fmt.Sprintf("'%s', `%s`.`%s`", fld.As, fld.From.As, col.Name))
				}
				selectExprs = append(selectExprs, fmt.Sprintf("JSON_MERGE_PATCH(`%s`.`%s`, JSON_OBJECT(%s))", from.As, jsql.SOURCE, strings.Join(columnExprs, ", ")))
			}
			selectExprs = append([]string{}, fmt.Sprintf("%s AS `%s`", strings.Join(selectExprs, ", "), jsql.RESULT))
		}
	} else {
		if len(query.Selects) == 0 {
			selectExprs = append(selectExprs, "*")
		}
	}

	return selectExprs
}

/**
* Query: Generates the SQL SELECT string for the given Query descriptor (MySQL dialect).
* @param query *jsql.Query
* @return string, error
**/
func (s *MySQL) Query(query *jsql.Query) (string, error) {
	if len(query.Froms) == 0 {
		return "", fmt.Errorf("query has no FROM source")
	}

	primary := query.Froms[0]

	var sb strings.Builder

	selects := mySelects(query)
	sb.WriteString("SELECT\n")
	sb.WriteString(strings.Join(selects, ",\n"))

	sb.WriteString(fmt.Sprintf("\nFROM `%s` AS `%s`", myFromRef(primary), primary.As))
	for _, from := range query.Froms[1:] {
		sb.WriteString(fmt.Sprintf(", `%s` AS `%s`", myFromRef(from), from.As))
	}

	for _, join := range query.Joins {
		sb.WriteString(fmt.Sprintf("\n%s `%s` AS `%s`", myJoinKeyword(join.Type), myFromRef(join.To), join.To.As))
		if len(join.Condition) > 0 {
			onSQL := myCondsSQL(query.GetField, query.UseSourceField, join.Condition, join.To.As)
			if onSQL != "" {
				sb.WriteString("\n  ON " + onSQL)
			}
		}
	}

	if len(query.Conditions) > 0 {
		whereSQL := myCondsSQL(query.GetField, query.UseSourceField, query.Conditions, primary.As)
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
			exprs = append(exprs, myFieldExpr(fld, query.UseSourceField))
		}
		sb.WriteString("\nGROUP BY " + strings.Join(exprs, ", "))
	}

	if len(query.Havings) > 0 {
		havingSQL := myCondsSQL(query.GetField, query.UseSourceField, query.Havings, primary.As)
		if havingSQL != "" {
			sb.WriteString("\nHAVING " + havingSQL)
		}
	}

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
			parts = append(parts, fmt.Sprintf("%s %s", myFieldExpr(fld, query.UseSourceField), dir))
		}
		sb.WriteString("\nORDER BY " + strings.Join(parts, ", "))
	}

	if query.Rows > 0 {
		sb.WriteString(fmt.Sprintf("\nLIMIT %d", query.Rows))
	}
	if query.Offset > 0 {
		sb.WriteString(fmt.Sprintf("\nOFFSET %d", query.Offset))
	}

	sb.WriteString(";")
	return sb.String(), nil
}
