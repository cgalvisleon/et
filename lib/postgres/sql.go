package lib

import (
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

// Return string can you use to select or return sql
func sqlColumns(l *linq.Linq, cols ...*linq.Lselect) string {
	if len(l.Froms) == 0 {
		return ""
	}

	var result string
	var def string

	appendColumn := func(val string) {
		result = strs.Append(result, val, ",\n")
	}

	appendColumns := func(f *linq.Lfrom, c *linq.Column) {
		if c.TypeColumn == linq.TpDetail {
			l.GetDetail(c)
		} else {
			s := l.GetColumn(c)
			if linq.TpColumn == c.TypeColumn {
				def = strs.Format(`%s`, s.As())
				appendColumn(def)
			} else if linq.TpAtrib == c.TypeColumn {
				def = strs.Format(`%s.%s#>>'{%s}'`, f.AS, linq.SourceField.Up(), c.Low())
				def = strs.Format(`%s AS %s`, def, c.Up())
				appendColumn(def)
			} else if linq.TpConcat == c.TypeColumn {
				def = c.Concat()
				def = strs.Format(`CONCAT(%s)`, def)
				def = strs.Format(`%s AS %s`, def, c.Up())
				appendColumn(def)
			} else if slices.Contains([]linq.TypeData{linq.TpRollup, linq.TpRelation}, c.TypeData) {
				r := c.RelationTo
				parent := l.NewFrom(r.Parent)
				def = strs.Format(`(SELECT row_to_json(%s) FROM %s AS %s WHERE %s LIMIT 1)`, r.SelectsAs(l), parent.Table(), parent.AS, r.WhereAs(l))
				def = strs.Format(`%s AS %s`, def, c.Up())
				appendColumn(def)
			} else if linq.TpFormula == c.TypeData {
				def = strs.Format(`(%s)`, c.Formula)
				def = strs.Format(`%s AS %s`, def, c.Up())
				appendColumn(def)
			}
		}
	}

	if len(cols) == 0 {
		f := l.Froms[0]

		for _, c := range f.Model.Columns {
			appendColumns(f, c)
		}
	}

	for _, c := range cols {
		appendColumns(c.From, c.Column)
	}

	return result
}

/**
* Return string can you use to select or return sql
* @param l *linq.Linq
* @param cols ...*linq.Lselect
* @return string
**/
func sqlData(l *linq.Linq, cols ...*linq.Lselect) string {
	if len(l.Froms) == 0 {
		return ""
	}

	var result string
	var objects string
	var def string
	var n int

	appendObjects := func(val string) {
		objects = strs.Append(objects, val, ",\n")
		n++
		if n >= 20 {
			def = strs.Format("jsonb_build_object(\n%s)", objects)
			result = strs.Append(result, def, "||")
			objects = ""
			n = 0
		}
	}

	appendColumns := func(f *linq.Lfrom, c *linq.Column) {
		if c.TypeColumn == linq.TpDetail {
			l.GetDetail(c)
		} else if !c.IsSourceField {
			s := l.GetColumn(c)
			if linq.TpColumn == c.TypeColumn { // 'name', A.NAME
				def = strs.Format(`'%s', %s`, c.Low(), s.As())
				appendObjects(def)
			} else if linq.TpAtrib == c.TypeColumn { // 'name', A._DATA#>>'{name}'
				if f.Linq.TypeQuery == linq.TpCommand {
					def = strs.Format(`%s#>>'{%s}'`, linq.SourceField.Up(), c.Low())
				} else {
					def = strs.Format(`%s.%s#>>'{%s}'`, f.AS, linq.SourceField.Up(), c.Low())
				}
				def = strs.Format(`'%s', %s`, c.Low(), def)
				appendObjects(def)
			} else if linq.TpConcat == c.TypeColumn {
				def = c.Concat()
				def = strs.Format(`CONCAT(%s)`, def)
				def = strs.Format(`'%s', %s`, c.Low(), def)
				appendObjects(def)
			} else if slices.Contains([]linq.TypeData{linq.TpRollup, linq.TpRelation}, c.TypeData) { // 'name', (SELECT row_to_json() FROM WUERE)
				r := c.RelationTo
				parent := l.NewFrom(r.Parent)
				def = strs.Format(`(SELECT row_to_json(%s) FROM %s AS %s WHERE %s LIMIT 1)`, r.SelectsAs(l), parent.Table(), parent.AS, r.WhereAs(l))
				def = strs.Format(`'%s', %s`, c.Low(), def)
				appendObjects(def)
			} else if linq.TpFormula == c.TypeData {
				def = strs.Format(`'%s', %s`, c.Low(), def)
				def = strs.Format(`(%s)`, c.Formula)
				appendObjects(def)
			}
		}
	}

	if len(cols) == 0 {
		f := l.Froms[0]
		m := f.Model
		for _, c := range m.Columns {
			appendColumns(f, c)
		}
		if n > 0 {
			def = strs.Format("jsonb_build_object(\n%s)", objects)
			result = strs.Append(result, def, "||")
		}

		return strs.Format(`%s AS %s`, result, linq.SourceField.Up())
	}

	for _, c := range cols {
		appendColumns(c.From, c.Column)
	}
	if n > 0 {
		def = strs.Format("jsonb_build_object(\n%s)", objects)
		result = strs.Append(result, def, "||")
	}

	return strs.Format(`%s AS %s`, result, linq.SourceField.Up())
}

// Add select to sql
func sqlSelect(l *linq.Linq) {
	var result string
	if l.Selects.Used {
		result = sqlColumns(l, l.Selects.Columns...)
	}
	if l.Data.Used {
		def := sqlData(l, l.Data.Columns...)
		result = strs.Append(result, def, ",\n")
	}

	if l.Distinct {
		result = strs.Append("SELECT DISTINCT", result, " ")
	} else {
		result = strs.Append("SELECT", result, " ")
	}

	l.Sql = strs.Append(l.Sql, result, "\n")
}

// Add from to sql
func sqlFrom(l *linq.Linq) error {
	if l.TypeQuery == linq.TpCommand {
		f := l.Command.From
		result := strs.Format(`FROM %s`, f.Model.Table)
		l.Sql = strs.Append(l.Sql, result, "\n")

		return nil
	}

	if len(l.Froms) == 0 {
		return logs.Errorm("From is required")
	}

	f := l.Froms[0]
	result := strs.Format(`FROM %s AS %s`, f.Model.Table, f.AS)
	l.Sql = strs.Append(l.Sql, result, "\n")

	return nil
}

// Add join to sql
func sqlJoin(l *linq.Linq) {
	var result string
	for _, v := range l.Joins {
		switch v.TypeJoin {
		case linq.Inner:
			result = strs.Format(`INNER JOIN %s AS %s ON %s`, v.T2.Table(), v.T2.AS, v.On.Where())
		case linq.Left:
			result = strs.Format(`LEFT JOIN %s AS %s ON %s`, v.T2.Table(), v.T2.AS, v.On.Where())
		case linq.Right:
			result = strs.Format(`RIGHT JOIN %s AS %s ON %s`, v.T2.Table(), v.T2.AS, v.On.Where())
		}
	}

	l.Sql = strs.Append(l.Sql, result, "\n")
}

// Add where to sql
func sqlWhere(l *linq.Linq) {
	var result string
	for i, v := range l.Wheres {
		if i == 0 {
			result = strs.Format(`WHERE %s`, v.Where())
		} else {
			def := strs.Format(`%s %s`, strs.Uppcase(v.Connetor), v.Where())
			result = strs.Append(result, def, "\n")
		}
	}

	l.Sql = strs.Append(l.Sql, result, "\n")
}

// Add group by to sql
func sqlGroupBy(l *linq.Linq) {
	var result string
	for i, v := range l.Groups {
		if i == 0 {
			result = strs.Format(`GROUP BY %s`, v.As())
		} else {
			result = strs.Append(result, v.As(), ", ")
		}
	}

	l.Sql = strs.Append(l.Sql, result, "\n")
}

// Add having to sql
func sqlHaving(l *linq.Linq) {
	var result string
	for i, v := range l.Havings {
		if i == 0 {
			result = strs.Format(`HAVING %s`, v.Where())
		} else {
			def := strs.Format(`%s %s`, strs.Uppcase(v.Connetor), v.Where())
			result = strs.Append(result, def, "\n")
		}
	}

	l.Sql = strs.Append(l.Sql, result, "\n")
}

// Add order by to sql
func sqlOrderBy(l *linq.Linq) {
	var result string
	for i, v := range l.Orders {
		if i == 0 {
			result = strs.Format(`ORDER BY %s %s`, v.As(), v.Sorted())
		} else {
			def := strs.Format(`%s %s`, v.As(), v.Sorted())
			result = strs.Append(result, def, ", ")
		}
	}

	l.Sql = strs.Append(l.Sql, result, "\n")
}

// Add limit to sql
func sqlLimit(l *linq.Linq) {
	if l.TypeQuery != linq.TpQuery {
		return
	}

	var result string
	if l.Limit > 0 {
		result = strs.Format(`LIMIT %d`, l.Limit)
	}

	l.Sql = strs.Append(l.Sql, result, "\n")
}

// Add offset to sql
func sqlOffset(l *linq.Linq) {
	if l.TypeQuery != linq.TpPage {
		return
	}

	var result string
	if l.Limit > 0 {
		result = strs.Format(`LIMIT %d OFFSET %d`, l.Limit, l.Offset)
	}

	l.Sql = strs.Append(l.Sql, result, "\n")
}

// Add return to sql
func sqlReturns(l *linq.Linq) {
	if !l.Returns.Used {
		return
	}

	var def, result string
	f := l.Froms[0]
	m := f.Model
	if m.UseSource {
		def = sqlData(l, l.Returns.Columns...)
	} else {
		def = sqlColumns(l, l.Returns.Columns...)
	}
	result = strs.Format(`RETURNING %s`, def)

	l.Sql = strs.Append(l.Sql, result, "\n")
}

// Build current sql used to trigger in linq
func sqlCurrent(l *linq.Linq) {
	var result string
	if l.Command.From.Model.UseSource {
		def := sqlData(l, []*linq.Lselect{}...)
		result = strs.Append(result, def, ",\n")
	} else {
		result = sqlColumns(l, []*linq.Lselect{}...)
	}

	result = strs.Append("SELECT", result, " ")

	l.Sql = strs.Append(l.Sql, result, "\n")
}

// Add sql insert to sql
func sqlInsert(l *linq.Linq) {
	var result string
	var columns string
	var values string
	var atribs string
	com := l.Command
	m := com.From.Model

	for k, v := range *com.Columns {
		if strs.Lowcase(k) == linq.SourceField.Low() {
			continue
		}

		field := strs.Uppcase(k)
		value := et.Unquote(v)
		def := strs.Format(`%v`, value)

		columns = strs.Append(columns, field, ", ")
		values = strs.Append(values, def, ", ")
	}

	for k, v := range *com.Atribs {
		field := strs.Lowcase(k)
		value := et.Quote(v)
		def := strs.Format(`"%s": %v`, field, value)

		atribs = strs.Append(atribs, def, ",\n")
	}

	if len(atribs) > 0 {
		columns = strs.Append(columns, linq.SourceField.Up(), ", ")
		def := strs.Format(`'{%s}'`, atribs)
		values = strs.Append(values, def, ", ")
	}

	result = strs.Format("INSERT INTO %s(%s)\nVALUES (%s)", m.Table, columns, values)

	l.Sql = strs.Append(l.Sql, result, "\n")
}

// Add sql update to sql
func sqlUpdate(l *linq.Linq) {
	var result string
	var values string
	var atribs string = linq.SourceField.Up()
	com := l.Command

	for k, v := range *com.Columns {
		field := strs.Uppcase(k)
		value := et.Unquote(v)
		def := strs.Format(`%s = %v`, field, value)

		values = strs.Append(values, def, ",\n")
	}

	for k, v := range *com.Atribs {
		field := strs.Lowcase(k)
		value := et.Quote(v)

		atribs = strs.Format(`jsonb_set(%s, '{%s}', '%v', true)`, atribs, field, value)
	}

	if len(atribs) > 0 {
		def := strs.Format(`%s = %v`, linq.SourceField.Up(), atribs)
		values = strs.Append(values, def, ",\n")
	}

	result = strs.Format("UPDATE %s SET\n%s", com.From.Model.Table, values)

	l.Sql = strs.Append(l.Sql, result, "\n")
}

// Add sql delete to sql
func sqlDelete(l *linq.Linq) {
	var result string
	com := l.Command

	result = strs.Format("DELETE FROM %s", com.From.Model.Table)

	l.Sql = strs.Append(l.Sql, result, "\n")
}
