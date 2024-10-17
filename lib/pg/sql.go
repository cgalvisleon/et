package lib

import (
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

/**
*sqlSelect add select to sql
* @param l *linq.Linq
**/
func sqlSelect(l *linq.Linq) {
	var result string
	if l.Selects.Used {
		def := sqlColumns(l, l.Selects.Columns...)
		if l.Selects.Distinct {
			result = strs.Append("SELECT DISTINCT", def, " ")
		} else {
			result = strs.Append("SELECT", def, " ")
		}
	} else if l.Datas.Used {
		def := sqlData(l, l.Datas.Columns...)
		def = strs.Append(result, def, ",\n")
		if l.Selects.Distinct {
			result = strs.Append("SELECT DISTINCT", def, " ")
		} else {
			result = strs.Append("SELECT", def, " ")
		}
	}

	l.Sql = strs.Append(l.Sql, result, "\n")
}

/**
* sqlFrom add from to sql
* @param l *linq.Linq
**/
func sqlFrom(l *linq.Linq) error {
	if l.TypeQuery == linq.TpCommand {
		model := l.Values.Model
		result := strs.Format(`FROM %s`, model.Table)
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

/**
* sqlJoin add join to sql
* @param l *linq.Linq
**/
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

/**
* sqlWhere add where to sql
* @param l *linq.Linq
**/
func sqlWhere(l *linq.Linq) {
	var result string

	def := func(v *linq.Lwhere) string {
		if v.Operator == "IN" {
			value := v.Unquote()
			value = strs.Replace(value, "]'", ")")
			value = strs.Replace(value, "'[", "(")
			return strs.Format(`%s %s %s`, v.As(), v.Operator, value)
		}

		return v.Where()
	}

	for i, v := range l.Wheres {
		if i == 0 {
			result = strs.Format(`WHERE %s`, def(v))
		} else {
			def := strs.Format(`%s %s`, strs.Uppcase(v.Connetor), def(v))
			result = strs.Append(result, def, "\n")
		}
	}

	l.Sql = strs.Append(l.Sql, result, "\n")
}

/**
* sqlGroupBy add group by to sql
* @param l *linq.Linq
**/
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

/**
* sqlHaving add having to sql
* @param l *linq.Linq
**/
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

/**
* sqlOrderBy add order by to sql
* @param l *linq.Linq
**/
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

/**
* sqlLimit add limit to sql
* @param l *linq.Linq
**/
func sqlLimit(l *linq.Linq) {
	if l.TypeQuery == linq.TpPage {
		var result string
		if l.Limit < linq.MaxRows {
			result = strs.Format(`LIMIT %d OFFSET %d`, l.Limit, l.Offset)
		} else {
			result = strs.Format(`LIMIT %d OFFSET %d`, linq.MaxRows, l.Offset)
		}

		l.Sql = strs.Append(l.Sql, result, "\n")
		return
	}

	if l.TypeQuery != linq.TpQuery {
		return
	}

	var result string
	if l.Limit <= 0 {
		result = strs.Format(`LIMIT %d`, linq.MaxRows)
	} else if l.Limit < linq.MaxRows {
		result = strs.Format(`LIMIT %d`, l.Limit)
	} else {
		result = strs.Format(`LIMIT %d`, linq.MaxRows)
	}

	l.Sql = strs.Append(l.Sql, result, "\n")
}
