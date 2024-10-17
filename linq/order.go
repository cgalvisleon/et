package linq

import "github.com/cgalvisleon/et/et"

// OrderBy struct to use in linq
type Lorder struct {
	Linq   *Linq
	Column *Lselect
	Asc    bool
}

/**
* Describe
* @return Json
**/
func (l *Lorder) Describe() et.Json {
	return et.Json{
		"column": l.Column.Describe(),
		"asc":    l.Asc,
	}
}

/**
* As
* @return string
**/
func (l *Lorder) As() string {
	return l.Column.As()
}

/**
* Sorted
* @return string
**/
func (l *Lorder) Sorted() string {
	if l.Asc {
		return "ASC"
	}

	return "DESC"
}

/**
* OrderBy
* @param asc
* @param columns
**/
func (l *Linq) OrderBy(asc bool, columns ...*Column) *Linq {
	for _, column := range columns {
		s := l.GetColumn(column)

		order := &Lorder{
			Linq:   l,
			Column: s,
			Asc:    asc,
		}

		l.Orders = append(l.Orders, order)
	}

	return l
}

/**
* OrderBy
* @param columns
**/
func (l *Linq) OrderByDesc(columns ...*Column) *Linq {
	return l.OrderBy(false, columns...)
}
