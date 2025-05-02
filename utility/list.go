package utility

import "slices"

type List []interface{}

func NewList() List {
	return make(List, 0)
}

func (s *List) Add(val interface{}) {
	idx := slices.IndexFunc(*s, func(e interface{}) bool { return e == val })
	if idx == -1 {
		*s = append(*s, val)
	}
}

func (s *List) Remove(val interface{}) {
	idx := slices.IndexFunc(*s, func(e interface{}) bool { return e == val })
	if idx != -1 {
		*s = slices.Delete(*s, idx, 1)
	}
}

func (s *List) Contains(val interface{}) bool {
	return slices.ContainsFunc(*s, func(e interface{}) bool { return e == val })
}

func (s *List) IndexOf(val interface{}) int {
	return slices.IndexFunc(*s, func(e interface{}) bool { return e == val })
}

func (s *List) Size() int {
	return len(*s)
}
