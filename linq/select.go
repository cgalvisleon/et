package linq

import (
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
)

// Select struct to use in linq
type Lselect struct {
	Linq       *Linq
	From       *Lfrom
	Column     *Column
	AS         string
	TpCaculate TpCaculate
}

// Describe method to use in linq
func (l *Lselect) Describe() et.Json {
	return et.Json{
		"form":          l.From.Describe(),
		"column":        l.Column.Name,
		"type":          l.Column.TypeColumn.String(),
		"as":            l.AS,
		"typeCalculate": l.TpCaculate.String(),
	}
}

// As method to use set as name to column in linq
func (l *Lselect) SetAs(name string) *Lselect {
	l.AS = name

	return l
}

// As method to use set as name to column in linq
func (l *Lselect) As() string {
	if l.Linq.TypeQuery == TpCommand {
		switch l.TpCaculate {
		case TpCount:
			def := strs.Format(`%s`, l.AS)
			return strs.Format(`COUNT(%s)`, def)
		case TpSum:
			def := strs.Format(`%s`, l.AS)
			return strs.Format(`SUM(%s)`, def)
		case TpAvg:
			def := strs.Format(`%s`, l.AS)
			return strs.Format(`AVG(%s)`, def)
		case TpMax:
			def := strs.Format(`%s`, l.AS)
			return strs.Format(`MAX(%s)`, def)
		case TpMin:
			def := strs.Format(`%s`, l.AS)
			return strs.Format(`MIN(%s)`, def)
		default:
			if l.Column == nil {
				return strs.Format(`%s`, et.Unquote(l.AS))
			}

			return strs.Format(`%s`, l.AS)
		}
	}

	switch l.TpCaculate {
	case TpCount:
		def := strs.Format(`%s.%s`, l.From.AS, l.AS)
		return strs.Format(`COUNT(%s)`, def)
	case TpSum:
		def := strs.Format(`%s.%s`, l.From.AS, l.AS)
		return strs.Format(`SUM(%s)`, def)
	case TpAvg:
		def := strs.Format(`%s.%s`, l.From.AS, l.AS)
		return strs.Format(`AVG(%s)`, def)
	case TpMax:
		def := strs.Format(`%s.%s`, l.From.AS, l.AS)
		return strs.Format(`MAX(%s)`, def)
	case TpMin:
		def := strs.Format(`%s.%s`, l.From.AS, l.AS)
		return strs.Format(`MIN(%s)`, def)
	default:
		if l.Column == nil {
			return strs.Format(`%s`, et.Unquote(l.AS))
		}

		return strs.Format(`%s.%s`, l.From.AS, l.AS)
	}
}

// Details method to use in linq
func (l *Lselect) FuncDetail(data *et.Json) {
	if l.Column.FuncDetail == nil {
		return
	}

	l.Column.FuncDetail(l.Column, data)
}

// Add column to details
func (l *Linq) GetDetail(column *Column) *Lselect {
	for _, v := range l.Details.Columns {
		if v.Column == column {
			return v
		}
	}

	lform := l.From(column.Model)
	result := &Lselect{
		Linq:       l,
		From:       lform,
		Column:     column,
		AS:         column.Name,
		TpCaculate: TpShowOriginal,
	}
	l.Details.Columns = append(l.Details.Columns, result)
	l.Details.Used = len(l.Details.Columns) > 0

	return result
}

func (l *Linq) GetAtrib(column *Column) *Lselect {
	for _, v := range l.Atribs {
		if v.Column == column {
			return v
		}
	}

	var result *Lselect
	l.GetColumn(column.Model.ColumnSource)
	lform := l.From(column.Model)
	result = &Lselect{
		Linq:       l,
		From:       lform,
		Column:     column,
		AS:         column.Name,
		TpCaculate: TpShowOriginal,
	}
	l.Atribs = append(l.Atribs, result)

	return result
}

// Add column to columns
func (l *Linq) GetColumn(column *Column) *Lselect {
	for _, v := range l.Columns {
		if v.Column == column {
			return v
		}
	}

	var result *Lselect
	switch column.TypeColumn {
	case TpDetail:
		result = l.GetDetail(column)
	case TpAtrib:
		result = l.GetAtrib(column)
	default:
		lform := l.From(column.Model)
		result = &Lselect{
			Linq:       l,
			From:       lform,
			Column:     column,
			AS:         column.Name,
			TpCaculate: TpShowOriginal,
		}
	}
	l.Columns = append(l.Columns, result)

	return result
}

// Add column to select by name
func (l *Linq) GetSelect(model *Model, name string) *Lselect {
	column := Col(model, name)
	if column == nil {
		return nil
	}

	result := l.GetColumn(column)

	for _, v := range l.Selects.Columns {
		if v.Column == column {
			return v
		}
	}

	l.Selects.Columns = append(l.Selects.Columns, result)

	return result
}

// Add column to data by name
func (l *Linq) GetData(model *Model, name string) *Lselect {
	column := Col(model, name)
	if column == nil {
		return nil
	}

	result := l.GetColumn(column)

	for _, v := range l.Datas.Columns {
		if v.Column == column {
			return v
		}
	}

	l.Datas.Columns = append(l.Datas.Columns, result)

	return result
}

// Select columns to use in linq
func (m *Model) Select(sel ...interface{}) *Linq {
	l := From(m)
	l.Selects.Used = true

	return l.Select(sel...)
}

func (m *Model) Distinct() *Linq {
	l := m.Data()

	return l.Distinct()
}

// Select SourceField a linq with data
func (m *Model) Data(sel ...interface{}) *Linq {
	if m.ColumnSource == nil {
		return m.Select(sel...)
	}

	l := From(m)
	l.Datas.Used = m.ColumnSource != nil

	return l.Data(sel...)
}

func (m *Model) Count(col *Column, as string) *Linq {
	l := From(m)
	l.Count(col, as)

	return l
}

func (m *Model) Sum(col *Column, as string) *Linq {
	l := From(m)
	l.Sum(col, as)

	return l
}

func (m *Model) Avg(col *Column, as string) *Linq {
	l := From(m)
	l.Avg(col, as)

	return l
}

func (m *Model) Max(col *Column, as string) *Linq {
	l := From(m)
	l.Max(col, as)

	return l
}

func (m *Model) Min(col *Column, as string) *Linq {
	l := From(m)
	l.Min(col, as)

	return l
}

/**
* Column find column in module to linq
* @param name string
* @return *Column
**/
func (l *Linq) Column(col interface{}) *Column {
	switch v := col.(type) {
	case Column:
		result := Col(v.Model, v.Name)
		if result == nil {
			return nil
		}
		return result
	case *Column:
		result := Col(v.Model, v.Name)
		if result == nil {
			return nil
		}
		return result
	case string:
		sp := strings.Split(v, ".")
		if len(sp) == 3 {
			s := sp[0]
			n := sp[1]
			m := l.DB.Table(s, n)
			if m != nil {
				result := Col(m, sp[2])
				if result == nil {
					return nil
				}
				return result
			}
		} else if len(sp) == 2 {
			n := sp[0]
			m := l.DB.Model(n)
			if m != nil {
				result := Col(m, sp[1])
				if result == nil {
					return nil
				}
				return result
			}
		} else {
			m := l.Froms[0].Model
			result := Col(m, v)
			if result == nil {
				return nil
			}
			return result
		}
	}

	return nil
}

// Select  columns a query
func (l *Linq) Select(sel ...interface{}) *Linq {
	for _, col := range sel {
		switch v := col.(type) {
		case Column:
			l.GetSelect(v.Model, v.Name)
		case *Column:
			l.GetSelect(v.Model, v.Name)
		case string:
			sp := strings.Split(v, ".")
			if len(sp) == 3 {
				s := sp[0]
				n := sp[1]
				m := l.DB.Table(s, n)
				if m != nil {
					l.GetSelect(m, sp[2])
				}
			} else if len(sp) == 2 {
				n := sp[0]
				m := l.DB.Model(n)
				if m != nil {
					l.GetSelect(m, sp[1])
				}
			} else {
				m := l.Froms[0].Model
				l.GetSelect(m, v)
			}
		}
	}

	return l
}

// Select SourceField a linq with data
func (l *Linq) Data(sel ...interface{}) *Linq {
	for _, col := range sel {
		switch v := col.(type) {
		case Column:
			l.GetData(v.Model, v.Name)
		case *Column:
			l.GetData(v.Model, v.Name)
		case string:
			sp := strings.Split(v, ".")
			if len(sp) == 3 {
				s := sp[0]
				n := sp[1]
				m := l.DB.Table(s, n)
				if m != nil {
					l.GetData(m, sp[2])
				}
			} else if len(sp) == 2 {
				n := sp[0]
				m := l.DB.Model(n)
				if m != nil {
					l.GetData(m, sp[1])
				}
			} else {
				m := l.Froms[0].Model
				l.GetData(m, v)
			}
		}
	}

	return l
}

// Select distinct columns a query
func (l *Linq) Distinct() *Linq {
	if l.Selects.Used {
		l.Selects.Distinct = true
	} else if l.Datas.Used {
		l.Datas.Distinct = true
	}

	return l
}
