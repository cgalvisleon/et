package linq

import (
	"database/sql"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

/**
* TypeQuery, type of query
**/
type TypeQuery int

/**
* TypeQuery, type of query
**/
const (
	TpQuery TypeQuery = iota
	TpCommand
	TpAll
	TpLast
	TpSkip
	TpPage
)

/**
* TypeQuery, return string type of query
**/
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

/**
* query
* @parms db
* @parms sql
* @parms args
* @return *sql.Rows
* @return error
**/
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

/**
* Exec execute a command in the database
* @parms db
* @parms sql
* @parms args
* @return sql.Result
* @return error
**/
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

/**
* Query execute a query in the database
* @parms db
* @parms sql
* @parms args
* @return et.Items
* @return error
**/
func Query(db *sql.DB, sql string, args ...any) (et.Items, error) {
	rows, err := query(db, sql, args...)
	if err != nil {
		return et.Items{}, err
	}
	defer rows.Close()

	items := RowsItems(rows)

	return items, nil
}

/**
* QueryOne execute a query in the database and return one item
* @parms db
* @parms sql
* @parms args
* @return et.Item
* @return error
**/
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

/**
* Query execute a query in the database
* @parms db
* @parms sql
* @parms args
* @return et.Items
* @return error
**/
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

/**
* QueryOne execute a query in the database and return one item
* @parms db
* @parms sql
* @parms args
* @return et.Item
* @return error
**/
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

/**
* Query execute a query in the database
* @parms sql
* @parms args
* @return et.Items
* @return error
**/
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

	rows, err := query(l.Db.DB, _query)
	if err != nil {
		return et.Items{}, logs.Error(err)
	}
	defer rows.Close()

	result := RowsItems(rows)

	return result, nil
}

/**
* Querysource execute a query in the database
* @parms sql
* @parms args
* @return et.Items
* @return error
**/
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

	rows, err := query(l.Db.DB, _query)
	if err != nil {
		return et.Items{}, logs.Error(err)
	}
	defer rows.Close()

	var result et.Items = et.Items{}
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

/**
* Exec execute a command in the database
* @return et.Items
* @return error
**/
func (l *Linq) Exec() (et.Items, error) {
	if l.TypeQuery != TpCommand {
		return et.Items{}, logs.Alertm("The query is not a command")
	}

	c := l.Values
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
	case TpDelete:
		err := c.Delete()
		if err != nil {
			return et.Items{}, err
		}
	}

	return *l.Result, nil
}

/**
* ExecOne is a Exec function and return item
* @return et.Item
* @return error
**/
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

/**
* Go is a Exec function and return items
* @return et.Items
* @return error
**/
func (l *Linq) Go() (et.Items, error) {
	return l.Exec()
}

/**
* GoOne is a Exec function and return item
* @return et.Item
* @return error
**/
func (l *Linq) GoOne() (et.Item, error) {
	return l.ExecOne()
}

/**
* Query is a function to execute a query
* @return et.Items
* @return error
**/
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

/**
* QueryOne is a function to execute a query and return one item
* @return et.Item
* @return error
**/
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

/**
* Take is a function to select n element data
* @return et.Items
* @return error
**/
func (l *Linq) Take(n int) (et.Items, error) {
	l.Limit = n

	return l.Query()
}

/**
* Skip is a function to skip n element data
* @return et.Items
* @return error
**/
func (l *Linq) Skip(n int) (et.Items, error) {
	l.TypeQuery = TpSkip
	l.Limit = 1
	l.Offset = n

	return l.Query()
}

/**
* All is a function to select all data
* @return et.Items
* @return error
**/
func (l *Linq) All() (et.Items, error) {
	l.Limit = 0

	return l.Query()
}

/**
* Find is a function to select all data
* @return et.Items
* @return error
**/
func (l *Linq) Find() (et.Items, error) {
	return l.All()
}

/**
* First is a function to select first data
* @return et.Item
* @return error
**/
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

/**
* Last is a function to select last data
* @return et.Item
* @return error
**/
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

/**
* Page is a function to select page data
* @return et.Items
* @return error
**/
func (l *Linq) Page(page, rows int) (et.Items, error) {
	l.TypeQuery = TpPage
	offset := (page - 1) * rows
	l.Limit = rows
	l.Offset = offset

	return l.Query()
}

/**
* List is a function to select page data and return list
* @return et.List
* @return error
**/
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
