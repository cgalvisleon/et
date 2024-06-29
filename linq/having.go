package linq

// Where method to use in linq
func (l *Linq) Having(where *Lwhere) *Linq {
	where.setLinq(l)
	l.Havings = append(l.Havings, where)
	l.isHaving = true

	return l
}
