package linq

import (
	"strings"

	"github.com/cgalvisleon/et/strs"
)

func (l *Linq) Concat(cols ...interface{}) *Column {
	result := &Column{
		Name:       "CONCAT",
		TypeColumn: TpConcat,
		Concats:    make([]*Lselect, 0),
	}

	addConcat := func(model *Model, name string) {
		name = nAme(name)
		lform := l.GetFrom(model)
		column := COlumn(model, name)
		col := &Lselect{
			Linq:       l,
			From:       lform,
			Column:     column,
			AS:         column.Name,
			TpCaculate: TpShowOriginal,
		}

		result.Concats = append(result.Concats, col)
	}

	for _, col := range cols {
		switch v := col.(type) {
		case Column:
			addConcat(v.Model, v.Name)
		case *Column:
			addConcat(v.Model, v.Name)
		case string:
			sp := strings.Split(v, ".")
			if len(sp) > 1 {
				n := sp[0]
				m := l.Db.Model(n)
				if m != nil {
					addConcat(m, sp[1])
				}
			} else {
				m := l.Froms[0].Model
				addConcat(m, v)
			}
		}
	}

	return result
}

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

func (m *Model) Concat(sel ...interface{}) *Column {
	l := From(m)

	return l.Concat(sel...)
}
