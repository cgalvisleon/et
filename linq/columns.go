package linq

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
)

// Define type columns in linq
type Lcolumns struct {
	Used     bool
	Columns  []*Lselect
	Distinct bool
}

/**
* Describe return a json with the definition of the columns
* @return et.Json
**/
func (l *Lcolumns) Describe() et.Json {
	var columns []et.Json = []et.Json{}
	for _, c := range l.Columns {
		columns = append(columns, c.Describe())
	}

	return et.Json{
		"used":     l.Used,
		"columns":  columns,
		"distinct": l.Distinct,
	}
}

/**
* SetAs set the name of the column
* @param name string
* @return *Lselect
**/
func (l *Lcolumns) SetAs(name string) *Lselect {
	for _, c := range l.Columns {
		if c.AS == strs.Uppcase(name) {
			return c
		}
	}

	return nil
}

/**
* NewColumns return a new columns
* @return *Lcolumns
**/
func NewColumns() *Lcolumns {
	return &Lcolumns{
		Used:    false,
		Columns: []*Lselect{},
	}
}
