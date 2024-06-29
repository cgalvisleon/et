package linq

import (
	"github.com/cgalvisleon/et/et"
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
	dbs       []*Database
	schemas   []*Schema
	models    []*Model
)

// Define type columns in linq
type Lcolumns struct {
	Used    bool
	Columns []*Lselect
}

// Definition method to use in linq
func (l *Lcolumns) Definition() et.Json {
	var columns []et.Json = []et.Json{}
	for _, c := range l.Columns {
		columns = append(columns, c.Definition())
	}

	return et.Json{
		"used":    l.Used,
		"columns": columns,
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
	Db        *Database
	as        int
	Froms     []*Lfrom
	Columns   []*Lselect
	Atribs    []*Lselect
	Selects   *Lcolumns
	Data      *Lcolumns
	Returns   *Lcolumns
	Details   *Lcolumns
	Distinct  bool
	Wheres    []*Lwhere
	Groups    []*Lgroup
	Havings   []*Lwhere
	isHaving  bool
	Orders    []*Lorder
	Joins     []*Ljoin
	Union     []*Linq
	Limit     int
	Offset    int
	Command   *Lcommand
	TypeQuery TypeQuery
	Sql       string
	Result    *et.Items
	ItIsBuilt bool
	debug     bool
}

func (l *Linq) Definition() *et.Json {
	var froms []et.Json = []et.Json{}
	for _, f := range l.Froms {
		froms = append(froms, f.Definition())
	}

	var columns []et.Json = []et.Json{}
	for _, c := range l.Columns {
		columns = append(columns, c.Definition())
	}

	var atribs []et.Json = []et.Json{}
	for _, a := range l.Atribs {
		atribs = append(atribs, a.Definition())
	}

	var wheres []et.Json = []et.Json{}
	for _, w := range l.Wheres {
		wheres = append(wheres, w.Definition())
	}

	var groups []et.Json = []et.Json{}
	for _, g := range l.Groups {
		groups = append(groups, g.Definition())
	}

	var havings []et.Json = []et.Json{}
	for _, h := range l.Havings {
		havings = append(havings, h.Definition())
	}

	var orders []et.Json = []et.Json{}
	for _, o := range l.Orders {
		orders = append(orders, o.Definition())
	}

	var joins []et.Json = []et.Json{}
	for _, j := range l.Joins {
		joins = append(joins, j.Definition())
	}

	var unions []et.Json = []et.Json{}
	for _, u := range l.Union {
		unions = append(unions, *u.Definition())
	}

	return &et.Json{
		"as":        l.as,
		"froms":     froms,
		"columns":   columns,
		"atribs":    atribs,
		"selects":   l.Selects.Definition(),
		"data":      l.Data.Definition(),
		"returns":   l.Returns.Definition(),
		"details":   l.Details.Definition(),
		"distinct":  l.Distinct,
		"wheres":    wheres,
		"groups":    groups,
		"havings":   havings,
		"orders":    orders,
		"joins":     joins,
		"unions":    unions,
		"limit":     l.Limit,
		"offset":    l.Offset,
		"command":   l.Command.Definition(),
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
func (l *Linq) User(val et.Json) *Linq {
	l.Command.User = val

	return l
}

// Set project to linq
func (l *Linq) Project(val et.Json) *Linq {
	l.Command.Project = val

	return l
}

// Init linq
func init() {
	dbs = []*Database{}
	schemas = []*Schema{}
	models = []*Model{}
}
