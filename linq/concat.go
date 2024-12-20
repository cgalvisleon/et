package linq

import (
	"github.com/cgalvisleon/et/strs"
)

/**
* Concat struct to use in linq
* @param []interfase{}
* @return *Lwhere
**/
func Concat(cols ...interface{}) *Lwhere {
	result := &Lwhere{
		Columns:  []*Lselect{},
		Operator: "",
		Value:    "",
	}

	for _, col := range cols {
		switch v := col.(type) {
		case Column:
			result.Columns = append(result.Columns, &Lselect{
				Column: &v, AS: v.Name,
			})
		case *Column:
			result.Columns = append(result.Columns, &Lselect{
				Column: v, AS: v.Name,
			})
		case string:
			result.Columns = append(result.Columns, &Lselect{
				AS: v,
			})
		}
	}

	return result
}

/**
* Concat method to use in linq
* @param []interfase{}
* @return *Lwhere
**/
func (l *Linq) Concat(cols ...interface{}) *Lwhere {
	where := Concat(cols...)
	where.setLinq(l)
	l.Wheres = append(l.Wheres, where)

	return where
}

/**
* Concat method to use in linq
* @return string
**/
func (c *Column) Concat() string {
	var result string
	for i, v := range c.Concats {
		if i == 0 {
			result = v.As()
		} else {
			result = strs.Append(result, v.As(), ",")
		}
	}

	return result
}
