package linq

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
)

// As method to use in linq from return leter string
func getAs(linq *Linq) string {
	n := linq.as

	limit := 18251
	base := 26
	as := ""
	a := n % base
	b := n / base
	c := b / base

	if n >= limit {
		n = n - limit + 702
		a = n % base
		b = n / base
		c = b / base
		b = b / base
		a = 65 + a
		b = 65 + b - 1
		c = 65 + c - 1
		as = fmt.Sprintf(`A%c%c%c`, rune(c), rune(b), rune(a))
	} else if b > base {
		b = b / base
		a = 65 + a
		b = 65 + b - 1
		c = 65 + c - 1
		as = fmt.Sprintf(`%c%c%c`, rune(c), rune(b), rune(a))
	} else if b > 0 {
		a = 65 + a
		b = 65 + b - 1
		as = fmt.Sprintf(`%c%c`, rune(b), rune(a))
	} else {
		a = 65 + a
		as = fmt.Sprintf(`%c`, rune(a))
	}

	linq.as++

	return as
}

// From struct to use in linq
type Lfrom struct {
	Linq  *Linq
	Model *Model
	AS    string
}

// Describe method to use in linq
func (l *Lfrom) Describe() et.Json {
	model := ""
	if l.Model != nil {
		model = l.Model.Name
	}

	return et.Json{
		"model": model,
		"as":    l.AS,
	}
}

// As method to use set as name to from in linq
func (l *Lfrom) As(name string) *Lfrom {
	l.AS = name

	return l
}

// Return table name in linq
func (l *Lfrom) Table() string {
	return l.Model.Table
}

// Return as column name in linq
func (l *Lfrom) AsColumn(col *Column) string {
	if l.Model == col.Model {
		return strs.Format(`%s.%s`, l.AS, col.Name)
	}

	return col.Name
}

// Find column in module to linq
func (l *Lfrom) Column(name string) *Column {
	return l.Model.Column(name)
}

// Shortcut to column in module to linq
func (l *Lfrom) Col(name string) *Column {
	return l.Column(name)
}

// Shortcut to column in module to linq
func (l *Lfrom) C(name string) *Column {
	return l.Column(name)
}

func NewLinQ() *Linq {
	return &Linq{
		Froms:     []*Lfrom{},
		Columns:   []*Lselect{},
		Atribs:    []*Lselect{},
		Selects:   NewColumns(),
		Datas:     NewColumns(),
		Returns:   NewColumns(),
		Details:   NewColumns(),
		Wheres:    []*Lwhere{},
		Groups:    []*Lgroup{},
		Orders:    []*Lorder{},
		Joins:     []*Ljoin{},
		Limit:     30,
		Offset:    0,
		TypeQuery: TpQuery,
		Sql:       "",
		Result:    &et.Item{},
		sets:      et.Json{},
	}
}

// From method new linq
func From(model *Model, as ...string) *Linq {
	result := NewLinQ()
	var aS string
	if len(as) > 0 {
		aS = as[0]
	} else {
		aS = getAs(result)
	}
	result.DB = model.DB
	form := &Lfrom{Linq: result, Model: model, AS: aS}
	result.Froms = append(result.Froms, form)
	result.Values = newValues(form, Tpnone)
	result.Values.Linq = result

	return result
}

// Get index model in linq
func (l *Linq) indexFrom(model *Model) int {
	result := -1
	for i, f := range l.Froms {
		if f.Model == model {
			result = i
			break
		}
	}

	return result
}

// From method to use in linq
func (l *Linq) From(model *Model, as ...string) *Lfrom {
	var result *Lfrom
	if len(as) > 0 {
		aS := as[0]
		result = &Lfrom{Linq: l, Model: model, AS: aS}
		l.Froms = append(l.Froms, result)

		return result
	}

	idx := l.indexFrom(model)
	if idx == -1 {
		aS := getAs(l)
		result = &Lfrom{Linq: l, Model: model, AS: aS}
		l.Froms = append(l.Froms, result)
	} else {
		result = l.Froms[idx]
	}

	return result
}

func (l *Linq) Set(values et.Json) *Linq {
	l.sets = values

	return l
}

func (l *Linq) F(as string) *Lfrom {
	result := -1
	for i, f := range l.Froms {
		if f.AS == as {
			result = i
			break
		}
	}

	if result == -1 {
		return nil
	}

	return l.Froms[result]
}
