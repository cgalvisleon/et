package linq

import (
	"database/sql"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

// TypeQuery struct to use in linq
type TypeQuery int

// Values for TypeQuery
const (
	TpQuery TypeQuery = iota
	TpCommand
	TpAll
	TpLast
	TpSkip
	TpPage
)

// String method to use in linq
func (d TypeQuery) String() string {
	switch d {
	case TpQuery:
		return "select"
	case TpCommand:
		return "command"
	case TpAll:
		return "all"
	case TpLast:
		return "last"
	case TpSkip:
		return "skip"
	case TpPage:
		return "page"
	}
	return ""
}

// Query execute a query in the database
func query(db *sql.DB, sql string, args ...any) (*sql.Rows, error) {
	if db == nil {
		return nil, logs.Alertm("Database is required")
	}

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// Exec execute a command in the database
func Exec(db *sql.DB, sql string, args ...any) (sql.Result, error) {
	if db == nil {
		return nil, logs.Alertm("Database is required")
	}

	result, err := db.Exec(sql, args...)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Query execute a query in the database
func Query(db *sql.DB, sql string, args ...any) (et.Items, error) {
	rows, err := query(db, sql, args...)
	if err != nil {
		return et.Items{}, err
	}
	defer rows.Close()

	items := RowsItems(rows)

	return items, nil
}

// QueryOne execute a query in the database and return one item
func QueryOne(db *sql.DB, sql string, args ...any) (et.Item, error) {
	items, err := Query(db, sql, args...)
	if err != nil {
		return et.Item{}, err
	}

	if items.Count == 0 {
		return et.Item{
			Ok:     false,
			Result: et.Json{},
		}, nil
	}

	return et.Item{
		Ok:     items.Ok,
		Result: items.Result[0],
	}, nil
}

// Query execute a query in the database
func (d *Database) Query(db *sql.DB, sql string, args ...any) (et.Items, error) {
	_query := SQLParse(sql, args...)

	if d.debug {
		logs.Debug(_query)
	}

	rows, err := query(db, _query)
	if err != nil {
		return et.Items{}, err
	}
	defer rows.Close()

	items := RowsItems(rows)

	return items, nil
}

// QueryOne execute a query in the database and return one item
func (d *Database) QueryOne(db *sql.DB, sql string, args ...any) (et.Item, error) {
	items, err := Query(db, sql, args...)
	if err != nil {
		return et.Item{}, err
	}

	if items.Count == 0 {
		return et.Item{
			Ok:     false,
			Result: et.Json{},
		}, nil
	}

	return et.Item{
		Ok:     items.Ok,
		Result: items.Result[0],
	}, nil
}

// Return sql row to items by linq
func (l *Linq) query(sql string, args ...any) (et.Items, error) {
	if l.Db.DB == nil {
		return et.Items{}, logs.Errorm("Connected is required")
	}

	if len(sql) == 0 {
		return et.Items{}, logs.Errorm("Sql is required")
	}

	_query := SQLParse(sql, args...)
	if l.debug {
		logs.Debug(l.Definition().ToString())
		logs.Debug(_query)
	}

	if !l.ItIsBuilt {
		return et.Items{}, logs.Alertm("Linq not built")
	}

	rows, err := query(l.Db.DB, _query)
	if err != nil {
		return et.Items{}, logs.Error(err)
	}
	defer rows.Close()

	var result et.Items
	for rows.Next() {
		var item et.Item
		item.Scan(rows)
		for _, col := range l.Details.Columns {
			col.FuncDetail(&item.Result)
		}

		result.Result = append(result.Result, item.Result)
		result.Ok = true
		result.Count++
	}

	return result, nil
}

// Return sql row _date to items by linq
func (l *Linq) querySource(sql string, args ...any) (et.Items, error) {
	if l.Db.DB == nil {
		return et.Items{}, logs.Errorm("Connected is required")
	}

	if len(sql) == 0 {
		return et.Items{}, logs.Errorm("Sql is required")
	}

	_query := SQLParse(sql, args...)
	if l.debug {
		logs.Debug(l.Definition().ToString())
		logs.Debug(_query)
	}

	if !l.ItIsBuilt {
		return et.Items{}, logs.Alertm("Linq not built")
	}

	rows, err := query(l.Db.DB, _query)
	if err != nil {
		return et.Items{}, logs.Error(err)
	}
	defer rows.Close()

	var result et.Items
	for rows.Next() {
		var item et.Item
		item.Scan(rows)
		for _, col := range l.Details.Columns {
			col.FuncDetail(&item.Result)
		}

		result.Result = append(result.Result, item.Result.Json(SourceField.Low()))
		result.Ok = true
		result.Count++
	}

	return result, nil
}

// Exec method to use in linq
func (l *Linq) Exec() (et.Items, error) {
	if l.TypeQuery != TpCommand {
		return et.Items{}, logs.Alertm("The query is not a command")
	}

	c := l.Command
	switch c.TypeCommand {
	case TpInsert:
		err := c.Insert()
		if err != nil {
			return et.Items{}, err
		}
	case TpUpdate:
		err := c.Update()
		if err != nil {
			return et.Items{}, err
		}
	case TpUdsert:
		err := c.Upsert()
		if err != nil {
			return et.Items{}, err
		}
	case TpDelete:
		err := c.Delete()
		if err != nil {
			return et.Items{}, err
		}
	}

	return *l.Result, nil
}

// ExecOne method to use in linq
func (l *Linq) ExecOne() (et.Item, error) {
	items, err := l.Exec()
	if err != nil {
		return et.Item{}, err
	}

	if !items.Ok {
		return et.Item{}, nil
	}

	return et.Item{
		Ok:     items.Ok,
		Result: items.Result[0],
	}, nil
}

// Select query
func (l *Linq) Query() (et.Items, error) {
	var err error
	l.Sql, err = l.selectSql()
	if err != nil {
		return et.Items{}, err
	}

	result, err := l.query(l.Sql)
	if err != nil {
		return et.Items{}, err
	}

	l.Result = &result

	return result, nil
}

// Execute query and return item
func (l *Linq) QueryOne() (et.Item, error) {
	items, err := l.Query()
	if err != nil {
		return et.Item{}, err
	}

	if items.Count == 0 {
		return et.Item{
			Ok:     false,
			Result: et.Json{},
		}, nil
	}

	return et.Item{
		Ok:     items.Ok,
		Result: items.Result[0],
	}, nil
}

// Select query take n element data
func (l *Linq) Take(n int) (et.Items, error) {
	l.Limit = n

	return l.Query()
}

// Select skip n element data
func (l *Linq) Skip(n int) (et.Items, error) {
	l.TypeQuery = TpSkip
	l.Limit = 1
	l.Offset = n

	return l.Query()
}

// Select query all data
func (l *Linq) All() (et.Items, error) {
	l.Limit = 0

	return l.Query()
}

// Select query first data
func (l *Linq) First() (et.Item, error) {
	items, err := l.Take(1)
	if err != nil {
		return et.Item{}, err
	}

	if !items.Ok {
		return et.Item{}, nil
	}

	return et.Item{
		Ok:     items.Ok,
		Result: items.Result[0],
	}, nil
}

// Select query type last data
func (l *Linq) Last() (et.Item, error) {
	l.TypeQuery = TpLast
	items, err := l.Take(1)
	if err != nil {
		return et.Item{}, err
	}

	if !items.Ok {
		return et.Item{}, nil
	}

	return et.Item{
		Ok:     items.Ok,
		Result: items.Result[0],
	}, nil
}

// Select query type page data
func (l *Linq) Page(page, rows int) (et.Items, error) {
	l.TypeQuery = TpPage
	offset := (page - 1) * rows
	l.Limit = rows
	l.Offset = offset

	return l.Query()
}

// Select query list, include count, page and rows
func (l *Linq) List(page, rows int) (et.List, error) {
	l.TypeQuery = TpAll
	var err error
	l.Sql, err = l.selectSql()
	if err != nil {
		return et.List{}, err
	}

	item, err := l.QueryOne()
	if err != nil {
		return et.List{}, err
	}

	all := item.Int("count")

	items, err := l.Page(page, rows)
	if err != nil {
		return et.List{}, err
	}

	return items.ToList(all, page, rows), nil
}
