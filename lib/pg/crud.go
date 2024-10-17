package lib

import "github.com/cgalvisleon/et/linq"

/**
* SelectSql return the sql to select
* @param l *linq.Linq
* @return string
**/
func (d *Postgres) SelectSql(l *linq.Linq) string {
	l.Clear()

	sqlSelect(l)

	sqlFrom(l)

	sqlJoin(l)

	sqlWhere(l)

	sqlGroupBy(l)

	sqlHaving(l)

	sqlOrderBy(l)

	sqlLimit(l)

	return l.SQL()
}

/**
* CurrentSql return the sql to select
* @param l *linq.Linq
* @return string
**/
func (d *Postgres) CurrentSql(l *linq.Linq) string {
	l.Clear()

	sqlCurrent(l)

	sqlFrom(l)

	sqlWhere(l)

	sqlLimit(l)

	return l.SQL()
}

/**
* InsertSql return the sql to insert
* @param l *linq.Linq
* @return string
**/
func (d *Postgres) InsertSql(l *linq.Linq) string {
	l.Clear()

	sqlInsert(l)

	sqlReturns(l)

	return l.SQL()
}

/**
* UpdateSql return the sql to update
* @param l *linq.Linq
* @return string
**/
func (d *Postgres) UpdateSql(l *linq.Linq) string {
	l.Clear()

	sqlUpdate(l)

	sqlReturns(l)

	return l.SQL()
}

/**
* DeleteSql return the sql to delete
* @param l *linq.Linq
* @return string
**/
func (d *Postgres) DeleteSql(l *linq.Linq) string {
	l.Clear()

	sqlDelete(l)

	sqlReturns(l)

	return l.SQL()
}
