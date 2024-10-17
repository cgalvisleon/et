package lib

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/strs"
)

/**
* sqlCurrent return string with columns or data
* @param l *linq.Linq
*
 */
func sqlCurrent(l *linq.Linq) {
	var result string
	module := l.Values.Model
	if module.ColumnSource == nil {
		result = sqlColumns(l, l.Selects.Columns...)
	} else {
		result = sqlData(l, l.Datas.Columns...)
	}

	result = strs.Append("SELECT", result, " ")
	l.Sql = strs.Append(l.Sql, result, "\n")
}

/**
* sqlInsert add insert to sql
* @param l *linq.Linq
**/
func sqlInsert(l *linq.Linq) {
	var result string
	var columns string
	var values string
	var atribs string
	vals := *l.Values
	mod := vals.Model

	for _, val := range vals.Values {
		if val.Column.IsDataField {
			continue
		}

		switch val.Column.TypeColumn {
		case linq.TpColumn:
			col := val.Column.Up()
			val := et.Unquote(val.New)
			def := strs.Format(`%v`, val)
			columns = strs.Append(columns, col, ", ")
			values = strs.Append(values, def, ", ")
		case linq.TpAtrib:
			col := val.Column.Low()
			val := et.Quote(val.New)
			def := strs.Format(`"%s": %v`, col, val)
			atribs = strs.Append(atribs, def, ",\n")
		}
	}

	if len(atribs) > 0 {
		columns = strs.Append(columns, linq.SourceField.Up(), ", ")
		def := strs.Format(`'{%s}'`, atribs)
		values = strs.Append(values, def, ", ")
	}

	result = strs.Format("INSERT INTO %s(%s)\nVALUES (%s)", mod.Table, columns, values)

	l.Sql = strs.Append(l.Sql, result, "\n")
	l.Args = append(l.Args, values)
}

/**
* sqlUpdate add update to sql
* @param l *linq.Linq
**/
func sqlUpdate(l *linq.Linq) {
	var result string
	var values string
	var atribs string = linq.SourceField.Up()
	vals := *l.Values
	mod := vals.Model

	for _, val := range vals.Values {
		if val.Column.IsDataField {
			continue
		}

		switch val.Column.TypeColumn {
		case linq.TpColumn:
			col := val.Column.Up()
			val := et.Unquote(val.New)
			def := strs.Format(`%s = %v`, col, val)
			values = strs.Append(values, def, ",\n")
		case linq.TpAtrib:
			col := val.Column.Low()
			val := et.Quote(val.New)
			atribs = strs.Format(`jsonb_set(%s, '{%s}', '%v', true)`, atribs, col, val)
		}
	}

	if len(atribs) > 0 {
		def := strs.Format(`%s = %v`, linq.SourceField.Up(), atribs)
		values = strs.Append(values, def, ",\n")
	}

	idt := et.Unquote(l.Values.IdT)
	result = strs.Format("UPDATE %s SET\n%s WHERE _IDT = '%v'", mod.Table, values, idt)

	l.Sql = strs.Append(l.Sql, result, "\n")
}

/**
* sqlDelete add delete to sql
* @param l *linq.Linq
**/
func sqlDelete(l *linq.Linq) {
	var result string
	vals := *l.Values
	mod := vals.Model

	idt := et.Unquote(l.Values.IdT)
	result = strs.Format("DELETE FROM %s WHERE _IDT = '%v'", mod.Table, idt)

	l.Sql = strs.Append(l.Sql, result, "\n")
}

/**
* sqlReturns add returns to sql
* @param l *linq.Linq
**/
func sqlReturns(l *linq.Linq) {
	if !l.Returns.Used {
		return
	}

	var def, result string
	f := l.Froms[0]
	m := f.Model
	if m.ColumnSource == nil {
		def = sqlColumns(l, l.Returns.Columns...)
	} else {
		def = sqlData(l, l.Returns.Columns...)
	}
	result = strs.Format(`RETURNING %s`, def)

	l.Sql = strs.Append(l.Sql, result, "\n")
}
