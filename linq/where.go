package linq

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

/**
* Lwhere struct to use in linq
**/
type Lwhere struct {
	Linq     *Linq
	Columns  []*Lselect
	Operator string
	Value    interface{}
	Connetor string
}

/**
* Unquote method to use in linq
* @return string
**/
func (w *Lwhere) Unquote() string {
	switch v := w.Value.(type) {
	case *Column:
		s := w.Linq.GetColumn(v)
		return s.As()
	case Column:
		s := w.Linq.GetColumn(&v)
		return s.As()
	case *Lselect:
		return v.As()
	case Lselect:
		return v.As()
	default:
		val := utility.Unquote(v)
		return strs.Format(`%v`, val)
	}
}

/**
* Quote method to use in linq
* @return string
**/
func (w *Lwhere) Quote() string {
	switch v := w.Value.(type) {
	case *Column:
		s := w.Linq.GetColumn(v)
		return s.As()
	case Column:
		s := w.Linq.GetColumn(&v)
		return s.As()
	case *Lselect:
		return v.As()
	case Lselect:
		return v.As()
	default:
		val := utility.Quote(v)
		return strs.Format(`%v`, val)
	}
}

/**
* Describe method to use in linq
* @return et.Json
**/
func (w *Lwhere) Describe() et.Json {
	value := w.Unquote()

	var columns []string = make([]string, 0)
	for _, _select := range w.Columns {
		columns = append(columns, _select.As())
	}

	return et.Json{
		"columns":  columns,
		"operator": w.Operator,
		"value":    value,
		"connetor": w.Connetor,
	}
}

/**
* As method to use in linq
* @return string
**/
func (w *Lwhere) As() string {
	var result string
	for i, _select := range w.Columns {
		if i == 0 {
			result = _select.As()
		} else {
			result = strs.Format(`%s, %s`, result, _select.As())
		}
	}

	if len(w.Columns) > 1 {
		result = strs.Format(`CONCAT(%s)`, result)
	}

	return result
}

/**
* Where method to use in linq
* @return string
**/
func (w *Lwhere) Where() string {
	value := w.Unquote()

	return strs.Format(`%s %s %s`, w.As(), w.Operator, value)
}

/**
* setLinq method to use in linq
* @param *Linq
* @return *Lwhere
**/
func (w *Lwhere) setLinq(l *Linq) *Lwhere {
	w.Linq = l
	for _, _select := range w.Columns {
		_select.Linq = l

		if _select.Column == nil {
			continue
		}
		if _select.Column.Model == nil {
			continue
		}

		_from := l.From(_select.Column.Model)
		_select.From = _from
	}

	return w
}

/**
* Eq method to use in linq
* @param interface{}
* @return *Lwhere
**/
func (w *Lwhere) Eq(val interface{}) *Lwhere {
	w.Operator = "="
	w.Value = val

	return w
}

/**
* Neg method to use in linq
* @param interface{}
* @return *Lwhere
**/
func (w *Lwhere) Neg(val interface{}) *Lwhere {
	w.Operator = "!="
	w.Value = val

	return w
}

/**
* In method to use in linq
* @param ...interface{}
* @return *Lwhere
**/
func (w *Lwhere) In(vals ...interface{}) *Lwhere {
	w.Operator = "IN"
	w.Value = vals

	return w
}

/**
* Like method to use in linq
* @param ...interface{}
* @return *Lwhere
**/
func (w *Lwhere) Like(val interface{}) *Lwhere {
	w.Operator = "LIKE"
	w.Value = val

	return w
}

/**
* More method to use in linq
* @param ...interface{}
* @return *Lwhere
**/
func (w *Lwhere) More(val interface{}) *Lwhere {
	w.Operator = ">"
	w.Value = val

	return w
}

/**
* Less method to use in linq
* @param ...interface{}
* @return *Lwhere
**/
func (w *Lwhere) Less(val interface{}) *Lwhere {
	w.Operator = "<"
	w.Value = val

	return w
}

/**
* MoreEq method to use in linq
* @param ...interface{}
* @return *Lwhere
**/
func (w *Lwhere) MoreEq(val interface{}) *Lwhere {
	w.Operator = ">="
	w.Value = val

	return w
}

/**
* LessEq method to use in linq
* @param ...interface{}
* @return *Lwhere
**/
func (w *Lwhere) LessEq(val interface{}) *Lwhere {
	w.Operator = "<="
	w.Value = val

	return w
}

/**
* Between method to use in linq
* @param ...interface{}
* @return *Lwhere
**/
func (w *Lwhere) Between(vals ...interface{}) *Lwhere {
	w.Operator = "BETWEEN"
	w.Value = vals

	return w
}

/**
* NotBetween method to use in linq
* @param ...interface{}
* @return *Lwhere
**/
func (w *Lwhere) NotBetween(vals ...interface{}) *Lwhere {
	w.Operator = "NOT BETWEEN"
	w.Value = vals

	return w
}

/**
* Where method to use in linq
* @param *Column
* @param string
* @param interface{}
* @return *Lwhere
**/
func Where(column *Column, operator string, value interface{}) *Lwhere {
	_select := &Lselect{Column: column, AS: column.Name}
	return &Lwhere{
		Columns:  []*Lselect{_select},
		Operator: operator,
		Value:    value,
	}
}

/**
* Where method to use in linq
* @param *Lselect
* @param string
* @param interface{}
* @return *Lwhere
**/
func (l *Linq) Where(where *Lwhere) *Linq {
	where.setLinq(l)
	l.Wheres = append(l.Wheres, where)
	l.isHaving = false

	return l
}

/**
* And connector to use in where
* @param *Lwhere
* @return *Linq
**/
func (l *Linq) And(where *Lwhere) *Linq {
	where.setLinq(l)
	where.Connetor = "AND"
	if l.isHaving {
		l.Havings = append(l.Havings, where)
	} else {
		l.Wheres = append(l.Wheres, where)
	}

	return l
}

/**
* Or connector to use in where
* @param *Lwhere
* @return *Linq
**/
func (l *Linq) Or(where *Lwhere) *Linq {
	where.setLinq(l)
	where.Connetor = "OR"
	if l.isHaving {
		l.Havings = append(l.Havings, where)
	} else {
		l.Wheres = append(l.Wheres, where)
	}

	return l
}

/**
* Eq method to use in column
* @param interface{}
* @return *Lwhere
**/
func (c *Column) Eq(val interface{}) *Lwhere {
	return Where(c, "=", val)
}

/**
* Neg method to use in column
* @param interface{}
* @return *Lwhere
**/
func (c *Column) Neg(val interface{}) *Lwhere {
	return Where(c, "!=", val)
}

/**
* In method to use in column
* @param ...interface{}
* @return *Lwhere
**/
func (c *Column) In(vals ...interface{}) *Lwhere {
	return Where(c, "IN", vals)
}

/**
* Like method to use in column
* @param ...interface{}
* @return *Lwhere
**/
func (c *Column) Like(val interface{}) *Lwhere {
	return Where(c, "LIKE", val)
}

/**
* More method to use in column
* @param ...interface{}
* @return *Lwhere
**/
func (c *Column) More(val interface{}) *Lwhere {
	return Where(c, ">", val)
}

/**
* Less method to use in column
* @param ...interface{}
* @return *Lwhere
**/
func (c *Column) Less(val interface{}) *Lwhere {
	return Where(c, ">", val)
}

/**
* MoreEq method to use in column
* @param ...interface{}
* @return *Lwhere
**/
func (c *Column) MoreEq(val interface{}) *Lwhere {
	return Where(c, ">=", val)
}

/**
* LessEq method to use in column
* @param ...interface{}
* @return *Lwhere
**/
func (c *Column) LessEq(val interface{}) *Lwhere {
	return Where(c, "<=", val)
}

/**
* Between method to use in column
* @param ...interface{}
* @return *Lwhere
**/
func (c *Column) Between(vals ...interface{}) *Lwhere {
	return Where(c, "BETWEEN", vals)
}

/**
* NotBetween method to use in column
* @param ...interface{}
* @return *Lwhere
**/
func (c *Column) NotBetween(vals ...interface{}) *Lwhere {
	return Where(c, "NOT BETWEEN", vals)
}
