package linq

import (
	"strings"
)

func (l *Linq) GetRetun(model *Model, name string) *Lselect {
	column := Col(model, name)
	if column == nil {
		return nil
	}

	for _, v := range l.Returns.Columns {
		if v.Column == column {
			return v
		}
	}

	result := l.GetColumn(column)
	l.Returns.Columns = append(l.Returns.Columns, result)
	l.Returns.Used = len(l.Returns.Columns) > 0

	return result
}

func (l *Linq) Returning(sel ...any) *Linq {
	for _, col := range sel {
		switch v := col.(type) {
		case Column:
			l.GetRetun(v.Model, v.Name)
		case *Column:
			l.GetRetun(v.Model, v.Name)
		case string:
			sp := strings.Split(v, ".")
			if len(sp) > 1 {
				n := sp[0]
				m := l.DB.Model(n)
				if m != nil {
					l.GetRetun(m, sp[1])
				}
			} else {
				m := l.Froms[0].Model
				l.GetRetun(m, v)
			}
		}
	}

	return l
}
