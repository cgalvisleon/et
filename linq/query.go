package linq

import (
	"database/sql"
	"strings"

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
func query(db *DB, sql string, args ...any) (*sql.Rows, error) {
	if db == nil {
		return nil, logs.Alertm("Database is required")
	}

	isSelect := func(query string) bool {
		query = strings.TrimSpace(query)
		return strings.HasPrefix(strings.ToLower(query), "select")
	}

	if !isSelect(sql) {
		return command(db, sql, args...)
	}

	rows, err := db.DB.Query(sql, args...)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

/**
* command
* @parms db
* @parms sql
* @parms args
* @return *sql.Rows
* @return error
**/
func command(db *DB, sql string, args ...any) (*sql.Rows, error) {
	if db == nil {
		return nil, logs.Alertm("Database is required")
	}

	query := SQLParse(sql, args...)
	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, err
	}

	driver := *db.Driver
	if !driver.Mode().IsNone() {
		err := driver.SetCommand(query)
		if err != nil {
			return nil, logs.Alert(err)
		}
	}

	return rows, nil
}

/**
* mutex
* @parms db
* @parms sql
* @parms args
* @return *sql.Rows
* @return error
**/
func mutex(db *DB, id, query string, index int64) error {
	if db == nil {
		return logs.Alertm("Database is required")
	}

	_, err := db.DB.Exec(query)
	if err != nil {
		return err
	}

	driver := *db.Driver
	if !driver.Mode().IsNone() {
		err := driver.SetMutex(id, query, index)
		if err != nil {
			return logs.Alert(err)
		}
	}

	return nil
}

/**
* Query execute a query in the database
* @parms sql
* @parms args
* @return et.Items
* @return error
**/
func (l *Linq) query(sql string, args ...any) (et.Items, error) {
	if l.DB == nil {
		return et.Items{}, logs.Alertm(MSG_COMMAND_REQUIRED)
	}

	if len(sql) == 0 {
		return et.Items{}, logs.Alertm(MSG_SQL_REQUIRED)
	}

	l.Sql = SQLParse(sql, args...)
	if l.debug {
		debug(l)
	}

	if l.showModel {
		showModel(l)
	}

	items, err := l.DB.Query(l.Sql)
	if err != nil {
		return et.Items{}, err
	}

	return items, nil
}

/**
* data execute a query in the database
* @parms sql
* @parms args
* @return et.Items
* @return error
**/
func (l *Linq) data(sql string, args ...any) (et.Items, error) {
	if l.DB == nil {
		return et.Items{}, logs.Errorm(MSG_COMMAND_REQUIRED)
	}

	if len(sql) == 0 {
		return et.Items{}, logs.Errorm(MSG_SQL_REQUIRED)
	}

	l.Sql = SQLParse(sql, args...)
	if l.debug {
		debug(l)
	}

	if l.showModel {
		showModel(l)
	}

	items, err := l.DB.Data(SourceField.Low(), l.Sql)
	if err != nil {
		return et.Items{}, err
	}

	return items, nil
}

/**
* command execute a query in the database
* @parms sql
* @parms args
* @return et.Items
* @return error
**/
func (l *Linq) command(sql string, args ...any) (et.Item, error) {
	if l.DB == nil {
		return et.Item{}, logs.Alertm(MSG_COMMAND_REQUIRED)
	}

	if len(sql) == 0 {
		return et.Item{}, logs.Alertm(MSG_SQL_REQUIRED)
	}

	l.Sql = SQLParse(sql, args...)
	if l.debug {
		debug(l)
	}

	if l.showModel {
		showModel(l)
	}

	items, err := l.DB.Command(l.Sql)
	if err != nil {
		return et.Item{}, err
	}

	return items, nil
}

/**
* Exec execute a command in the database
* @return et.Items
* @return error
**/
func (l *Linq) Exec() (et.Item, error) {
	if l.TypeQuery != TpCommand {
		return et.Item{}, logs.Alertm(MSG_QUERY_NOT_COMMAND)
	}

	c := l.Values
	switch c.TypeCommand {
	case TpInsert:
		err := c.Insert()
		if err != nil {
			return et.Item{}, err
		}
	case TpUpdate:
		err := c.Update()
		if err != nil {
			return et.Item{}, err
		}
	case TpDelete:
		err := c.Delete()
		if err != nil {
			return et.Item{}, err
		}
	}

	return *l.Result, nil
}

/**
* Go is a Exec function and return items
* @return et.Items
* @return error
**/
func (l *Linq) Go() (et.Item, error) {
	return l.Exec()
}

/**
* Query is a function to execute a query
* @return et.Items
* @return error
**/
func (l *Linq) Query() (et.Items, error) {
	var err error
	var items et.Items

	l.Sql, err = l.selectSql()
	if err != nil {
		return et.Items{}, err
	}

	if l.Datas.Used {
		items, err = l.data(l.Sql)
		if err != nil {
			return et.Items{}, err
		}
	} else {
		items, err = l.query(l.Sql)
		if err != nil {
			return et.Items{}, err
		}
	}

	for _, data := range items.Result {
		for k, v := range l.sets {
			data.Set(k, v)
		}

		for _, col := range l.Details.Columns {
			col.FuncDetail(&data)
		}
	}

	return items, nil
}

/**
* QueryOne is a function to execute a query and return one item
* @return et.Item
* @return error
**/
func (l *Linq) One() (et.Item, error) {
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

	item, err := l.One()
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
