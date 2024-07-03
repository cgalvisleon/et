package linq

// Count function to use in linq
func (l *Linq) Count(col *Column, as string) *Linq {
	sel := l.GetColumn(col)
	sel.TpCaculate = TpCount
	if as == "" {
		sel.AS = "count"
	}

	return l
}

// Sum function to use in linq
func (l *Linq) Sum(col *Column, as string) *Linq {
	sel := l.GetColumn(col)
	sel.TpCaculate = TpSum
	if as == "" {
		sel.AS = "sum"
	}

	return l
}

// Avg function to use in linq
func (l *Linq) Avg(col *Column, as string) *Linq {
	sel := l.GetColumn(col)
	sel.TpCaculate = TpAvg
	if as == "" {
		sel.AS = "avg"
	}

	return l
}

// Max function to use in linq
func (l *Linq) Max(col *Column, as string) *Linq {
	sel := l.GetColumn(col)
	sel.TpCaculate = TpMax
	if as == "" {
		sel.AS = "max"
	}

	return l
}

// Min function to use in linq
func (l *Linq) Min(col *Column, as string) *Linq {
	sel := l.GetColumn(col)
	sel.TpCaculate = TpMin
	if as == "" {
		sel.AS = "min"
	}

	return l
}
