package linq

import "github.com/cgalvisleon/et/et"

type TypeJoin int

const (
	Inner TypeJoin = iota
	Left
	Right
)

func (d TypeJoin) String() string {
	switch d {
	case Inner:
		return "inner"
	case Left:
		return "left"
	case Right:
		return "right"
	}
	return ""
}

// Join struct to use in linq
type Ljoin struct {
	Linq     *Linq
	T1       *Lfrom
	T2       *Lfrom
	On       *Lwhere
	TypeJoin TypeJoin
}

// Describe method to use in linq
func (l *Ljoin) Describe() et.Json {
	return et.Json{
		"t1":   l.T1.Describe(),
		"t2":   l.T2.Describe(),
		"on":   l.On.Describe(),
		"type": l.TypeJoin.String(),
	}
}

func newJoin(l *Linq, t1, t2 *Model, where *Lwhere) *Ljoin {
	_t1 := l.From(t1)
	_t2 := l.From(t2)

	where = where.setLinq(l)

	switch v := where.Value.(type) {
	case *Column:
		_select := &Lselect{
			Linq:   l,
			From:   _t2,
			Column: v,
			AS:     v.Name,
		}
		where.Value = _select
	case *Lselect:
		v.Linq = l
		v.From = _t2
		v.AS = v.Column.Name
	}

	return &Ljoin{
		Linq:     l,
		T1:       _t1,
		T2:       _t2,
		On:       where,
		TypeJoin: Inner,
	}
}

// Inner Join method to use in linq
func (l *Linq) Join(t1, t2 *Model, where *Lwhere) *Linq {
	join := newJoin(l, t1, t2, where)
	l.Joins = append(l.Joins, join)

	return l
}

// Left Join method to use in linq
func (l *Linq) LeftJoin(t1, t2 *Model, where *Lwhere) *Linq {
	join := newJoin(l, t1, t2, where)
	join.TypeJoin = Left
	l.Joins = append(l.Joins, join)

	return l
}

// Right Join method to use in linq
func (l *Linq) RightJoin(t1, t2 *Model, where *Lwhere) *Linq {
	join := newJoin(l, t1, t2, where)
	join.TypeJoin = Right
	l.Joins = append(l.Joins, join)

	return l
}
