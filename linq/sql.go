package linq

// Return sql select by linq
func (l *Linq) selectSql() (string, error) {
	return l.DB.selectSql(l)
}

// Return sql current by linq
func (l *Linq) currentSql() (string, error) {
	return l.DB.currentSql(l)
}

// Return sql insert by linq
func (l *Linq) insertSql() (string, error) {
	return l.DB.insertSql(l)
}

// Return sql update by linq
func (l *Linq) updateSql() (string, error) {
	return l.DB.updateSql(l)
}

// Return sql delete by linq
func (l *Linq) deleteSql() (string, error) {
	return l.DB.deleteSql(l)
}
