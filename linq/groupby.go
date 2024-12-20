package linq

import "github.com/cgalvisleon/et/et"

// GroupBy struct to use in linq
type Lgroup struct {
	Linq   *Linq
	Column *Lselect
	AS     string
}

// Describe method to use in linq
func (l *Lgroup) Describe() et.Json {
	return et.Json{
		"column": l.Column.Describe(),
		"as":     l.AS,
	}
}

// As method to use set as name to column in linq
func (l *Lgroup) SetAs(name string) *Lgroup {
	l.AS = name

	return l
}

// As method to use set as name to column in linq
func (l *Lgroup) As() string {
	return l.Column.As()
}

// GroupBy method to use in linq
func (l *Linq) GroupBy(columns ...*Column) *Linq {
	for _, column := range columns {
		s := l.GetColumn(column)

		group := &Lgroup{
			Linq:   l,
			Column: s,
		}

		l.Groups = append(l.Groups, group)
	}

	return l
}
