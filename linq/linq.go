package linq

import (
	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/strs"
)

type TypeVar int

// Var field system name
const (
	IdTField TypeVar = iota
	SourceField
	IndexField
	StateField
)

// Return upcase to field system
func (t TypeVar) Up() string {
	switch t {
	case IdTField:
		return "_IDT"
	case SourceField:
		return "_DATA"
	case IndexField:
		return "_INDEX"
	case StateField:
		return "_STATE"
	}

	return ""
}

// Return lowcase to field system
func (t TypeVar) Low() string {
	switch t {
	case IdTField:
		return "_idt"
	case SourceField:
		return "_data"
	case IndexField:
		return "_index"
	case StateField:
		return "_state"
	}

	return ""
}

// Global variables
var (
	MaxUpdate = 1000
	MaxDelete = 1000
	MaxRows   = 1000
	dbs       []*Database
	schemas   []*Schema
	models    []*Model
)

// Define type columns in linq
type Lcolumns struct {
	Used     bool
	Columns  []*Lselect
	Distinct bool
}

// Describe method to use in linq
func (l *Lcolumns) Describe() js.Json {
	var columns []js.Json = []js.Json{}
	for _, c := range l.Columns {
		columns = append(columns, c.Describe())
	}

	return js.Json{
		"used":     l.Used,
		"columns":  columns,
		"distinct": l.Distinct,
	}
}

// As method to use set as name to column in linq
func (l *Lcolumns) SetAs(name string) *Lselect {
	for _, c := range l.Columns {
		if c.AS == strs.Uppcase(name) {
			return c
		}
	}

	return nil
}

func NewColumns() *Lcolumns {
	return &Lcolumns{
		Used:    false,
		Columns: []*Lselect{},
	}
}

// Linq struct
type Linq struct {
	DB        *Database
	Froms     []*Lfrom
	Columns   []*Lselect
	Atribs    []*Lselect
	Selects   *Lcolumns
	Datas     *Lcolumns
	Returns   *Lcolumns
	Details   *Lcolumns
	Wheres    []*Lwhere
	Groups    []*Lgroup
	Havings   []*Lwhere
	isHaving  bool
	Orders    []*Lorder
	Joins     []*Ljoin
	Union     []*Linq
	Limit     int
	Offset    int
	Values    *Values
	TypeQuery TypeQuery
	Sql       string
	Result    *js.Items
	as        int
	setResult js.Json
	debug     bool
}

/**
* Describe return a json with the definition of the linq
* @return js.Json
**/
func (l *Linq) Describe() *js.Json {
	var froms []js.Json = []js.Json{}
	for _, f := range l.Froms {
		froms = append(froms, f.Describe())
	}

	var columns []js.Json = []js.Json{}
	for _, c := range l.Columns {
		columns = append(columns, c.Describe())
	}

	var atribs []js.Json = []js.Json{}
	for _, a := range l.Atribs {
		atribs = append(atribs, a.Describe())
	}

	var wheres []js.Json = []js.Json{}
	for _, w := range l.Wheres {
		wheres = append(wheres, w.Describe())
	}

	var groups []js.Json = []js.Json{}
	for _, g := range l.Groups {
		groups = append(groups, g.Describe())
	}

	var havings []js.Json = []js.Json{}
	for _, h := range l.Havings {
		havings = append(havings, h.Describe())
	}

	var orders []js.Json = []js.Json{}
	for _, o := range l.Orders {
		orders = append(orders, o.Describe())
	}

	var joins []js.Json = []js.Json{}
	for _, j := range l.Joins {
		joins = append(joins, j.Describe())
	}

	var unions []js.Json = []js.Json{}
	for _, u := range l.Union {
		unions = append(unions, *u.Describe())
	}

	return &js.Json{
		"as":        l.as,
		"froms":     froms,
		"columns":   columns,
		"atribs":    atribs,
		"selects":   l.Selects.Describe(),
		"data":      l.Datas.Describe(),
		"returns":   l.Returns.Describe(),
		"details":   l.Details.Describe(),
		"wheres":    wheres,
		"groups":    groups,
		"havings":   havings,
		"orders":    orders,
		"joins":     joins,
		"unions":    unions,
		"limit":     l.Limit,
		"offset":    l.Offset,
		"values":    l.Values.Describe(),
		"typeQuery": l.TypeQuery.String(),
		"sql":       l.Sql,
	}
}

// AddSelect method to use in linq
func (l *Linq) Debug() *Linq {
	l.debug = true

	return l
}

// Add ';' to end sql and return
func (l *Linq) SQL() string {
	l.Sql = strs.Format(`%s;`, l.Sql)

	return l.Sql
}

// Clear sql
func (l *Linq) Clear() string {
	l.Sql = ""

	return l.Sql
}

// Set user to linq
func (l *Linq) User(val interface{}) *Linq {
	l.Values.User = val

	return l
}

// Set project to linq
func (l *Linq) Project(val interface{}) *Linq {
	l.Values.Project = val

	return l
}

// Init linq
func init() {
	dbs = []*Database{}
	schemas = []*Schema{}
	models = []*Model{}
}
