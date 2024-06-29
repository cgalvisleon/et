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

// Definition method to use in linq
func (l *Lselect) Definition() et.Json {
	return et.Json{
		"form":          l.From.Definition(),
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
		return strs.Format(`%s.%s`, l.From.AS, l.AS)
	}
}

// Details method to use in linq
func (l *Lselect) FuncDetail(data *et.Json) {
	l.Column.FuncDetail(l.Column, data)
}

// Add column to details
func (l *Linq) GetDetail(column *Column) *Lselect {
	for _, v := range l.Details.Columns {
		if v.Column == column {
			return v
		}
	}

	lform := l.GetFrom(column.Model)
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
	l.GetColumn(column.Model.Source)
	lform := l.GetFrom(column.Model)
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
	if column.TypeColumn == TpDetail {
		result = l.GetDetail(column)
	} else if column.TypeColumn == TpAtrib {
		result = l.GetAtrib(column)
	} else {
		lform := l.GetFrom(column.Model)
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
	column := COlumn(model, name)
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
	l.Selects.Used = len(l.Selects.Columns) > 0

	return result
}

// Add column to data by name
func (l *Linq) GetData(model *Model, name string) *Lselect {
	column := COlumn(model, name)
	if column == nil {
		return nil
	}

	result := l.GetColumn(column)

	for _, v := range l.Data.Columns {
		if v.Column == column {
			return v
		}
	}

	l.Data.Columns = append(l.Data.Columns, result)
	l.Data.Used = len(l.Data.Columns) > 0

	return result
}

// Select columns to use in linq
func (m *Model) Select(sel ...any) *Linq {
	l := From(m)

	return l.Select(sel...)
}

func (m *Model) Distint(sel ...any) *Linq {
	l := From(m)

	return l.DIstinct(sel...)
}

// Select SourceField a linq with data
func (m *Model) Data(sel ...any) *Linq {
	l := From(m)

	return l.DAta(sel...)
}

// Select  columns a query
func (l *Linq) Select(sel ...any) *Linq {
	l.Selects.Used = true

	for _, col := range sel {
		switch v := col.(type) {
		case Column:
			l.GetSelect(v.Model, v.Name)
		case *Column:
			l.GetSelect(v.Model, v.Name)
		case string:
			sp := strings.Split(v, ".")
			if len(sp) > 1 {
				n := sp[0]
				m := l.Db.Model(n)
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

// Select distinct columns a query
func (l *Linq) DIstinct(sel ...any) *Linq {
	l.Distinct = true

	return l.Select(sel...)
}

// Select SourceField a linq with data
func (l *Linq) DAta(sel ...any) *Linq {
	l.Data.Used = true

	for _, col := range sel {
		switch v := col.(type) {
		case Column:
			l.GetData(v.Model, v.Name)
		case *Column:
			l.GetData(v.Model, v.Name)
		case string:
			sp := strings.Split(v, ".")
			if len(sp) > 1 {
				n := sp[0]
				m := l.Db.Model(n)
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
