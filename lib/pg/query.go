package lib

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/strs"
)

/**
* sqlColumns return string with columns
* @param l *linq.Linq
* @param cols ...*linq.Lselect
* @return string
**/
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
			switch c.TypeData {
			case linq.TpFunction:
				val := et.Unquote(c.Default)
				def = strs.Format(`%s AS %s`, val, c.Up())
				appendColumn(def)
			case linq.TpFormula:
				def = strs.Format(`(%s)`, c.Formula)
				def = strs.Format(`%s AS %s`, def, c.Up())
				appendColumn(def)
			}
		} else {
			s := l.GetColumn(c)
			if c.TypeColumn == linq.TpColumn {
				def = strs.Format(`%s`, s.As())
				appendColumn(def)
			} else if c.TypeColumn == linq.TpAtrib {
				if f.Linq.TypeQuery == linq.TpCommand {
					def = strs.Format(`%s#>>'{%s}'`, linq.SourceField.Up(), c.Low())
				} else {
					def = strs.Format(`%s.%s#>>'{%s}'`, f.AS, linq.SourceField.Up(), c.Low())
				}
				def = strs.Format(`%s AS %s`, def, c.Up())
				appendColumn(def)
			} else if c.TypeColumn == linq.TpConcat {
				def = c.Concat()
				def = strs.Format(`CONCAT(%s)`, def)
				def = strs.Format(`%s AS %s`, def, c.Up())
				appendColumn(def)
			} else if c.TypeData == linq.TpRollup {
				r := c.RelationTo
				parent := l.From(r.Parent)
				def = strs.Format(`(SELECT %s FROM %s AS %s WHERE %s LIMIT 1)`, r.SelectsAs(l), parent.Table(), parent.AS, r.WhereAs(l))
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
* sqlData return string with data
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
			switch c.TypeData {
			case linq.TpFunction:
				val := et.Unquote(c.Default)
				def = strs.Format(`'%s', %s`, c.Low(), val)
				appendObjects(def)
			case linq.TpFormula:
				def = strs.Format(`'%s', %s`, c.Low(), def)
				def = strs.Format(`(%s)`, c.Formula)
				appendObjects(def)
			}
		} else if !c.IsDataField {
			s := l.GetColumn(c)
			if c.TypeColumn == linq.TpColumn { // 'name', A.NAME
				def = strs.Format(`'%s', %s`, c.Low(), s.As())
				appendObjects(def)
			} else if c.TypeColumn == linq.TpAtrib { // 'name', A._DATA#>>'{name}'
				if f.Linq.TypeQuery == linq.TpCommand {
					def = strs.Format(`%s#>>'{%s}'`, linq.SourceField.Up(), c.Low())
				} else {
					def = strs.Format(`%s.%s#>>'{%s}'`, f.AS, linq.SourceField.Up(), c.Low())
				}
				def = strs.Format(`'%s', %s`, c.Low(), def)
				appendObjects(def)
			} else if c.TypeColumn == linq.TpConcat {
				def = c.Concat()
				def = strs.Format(`CONCAT(%s)`, def)
				def = strs.Format(`'%s', %s`, c.Low(), def)
				appendObjects(def)
			} else if c.TypeData == linq.TpRollup {
				r := c.RelationTo
				parent := l.From(r.Parent)
				def = strs.Format(`(SELECT %s FROM %s AS %s WHERE %s LIMIT 1)`, r.SelectsAs(l), parent.Table(), parent.AS, r.WhereAs(l))
				def = strs.Format(`'%s', %s`, c.Low(), def)
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
