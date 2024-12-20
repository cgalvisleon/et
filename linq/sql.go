package linq

import "github.com/cgalvisleon/et/strs"

/**
* SQL return sql
* @return string
**/
func (l *Linq) SQL() string {
	l.Sql = strs.Format(`%s;`, l.Sql)

	return l.Sql
}

/**
* Clear clear sql
* @return string
**/
func (l *Linq) Clear() string {
	l.Sql = ""

	return l.Sql
}

/**
* selectSql return sql select by linq
* @return string
* @return error
**/
func (l *Linq) selectSql() (string, error) {
	return l.DB.selectSql(l)
}

/**
* currentSql return sql current by linq
* @return string
* @return error
**/
func (l *Linq) currentSql() (string, error) {
	return l.DB.currentSql(l)
}

/**
* insertSql return sql insert by linq
* @return string
* @return error
**/
func (l *Linq) insertSql() (string, error) {
	return l.DB.insertSql(l)
}

/**
* updateSql return sql update by linq
* @return string
* @return error
**/
func (l *Linq) updateSql() (string, error) {
	return l.DB.updateSql(l)
}

/**
* deleteSql return sql delete by linq
* @return string
* @return error
**/
func (l *Linq) deleteSql() (string, error) {
	return l.DB.deleteSql(l)
}
