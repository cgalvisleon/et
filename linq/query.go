package linq

import (
	"database/sql"

	"github.com/cgalvisleon/et/js"
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

func exec(db *sql.DB, sql string, args ...any) (*sql.Rows, error) {
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
* Query execute a query in the database
* @parms db
* @parms sql
* @parms args
* @return js.Items
* @return error
**/
func Query(db *sql.DB, sql string, args ...any) (js.Items, error) {
	rows, err := query(db, sql, args...)
	if err != nil {
		return js.Items{}, err
	}
	defer rows.Close()

	result := RowsToItems(rows)

	return result, nil
}

/**
* QueryOne execute a query in the database and return one item
* @parms db
* @parms sql
* @parms args
* @return js.Item
* @return error
**/
func QueryOne(db *sql.DB, sql string, args ...any) (js.Item, error) {
	rows, err := query(db, sql, args...)
	if err != nil {
		return js.Item{}, err
	}
	defer rows.Close()

	result := RowsToItem(rows)

	return result, nil
}

/**
* Exec execute a command in the database
* @parms db
* @parms sql
* @parms args
* @return sql.Result
* @return error
**/
func Exec(db *sql.DB, sql string, args ...any) (js.Item, error) {
	rows, err := exec(db, sql, args...)
	if err != nil {
		return js.Item{}, err
	}
	defer rows.Close()

	result := RowsToItem(rows)

	return result, nil
}

/**
* Query execute a query in the database
* @parms sql
* @parms args
* @return js.Items
* @return error
**/
func (l *Linq) query(sql string, args ...any) (js.Items, error) {
	if l.DB == nil {
		return js.Items{}, logs.Alertm("Connected is required")
	}

	if len(sql) == 0 {
		return js.Items{}, logs.Alertm("Sql is required")
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
		return js.Items{}, err
	}

	return items, nil
}

/**
* data execute a query in the database
* @parms sql
* @parms args
* @return js.Items
* @return error
**/
func (l *Linq) data(sql string, args ...any) (js.Items, error) {
	if l.DB == nil {
		return js.Items{}, logs.Errorm("Connected is required")
	}

	if len(sql) == 0 {
		return js.Items{}, logs.Errorm("Sql is required")
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
		return js.Items{}, err
	}

	return items, nil
}

/**
* exec execute a query in the database
* @parms sql
* @parms args
* @return js.Items
* @return error
**/
func (l *Linq) exec(sql string, args ...any) (js.Item, error) {
	if l.DB == nil {
		return js.Item{}, logs.Errorm("Connected is required")
	}

	if len(sql) == 0 {
		return js.Item{}, logs.Errorm("Sql is required")
	}

	l.Sql = SQLParse(sql, args...)
	if l.debug {
		debug(l)
	}

	if l.showModel {
		showModel(l)
	}

	result, err := l.DB.Exec(sql, args...)
	if err != nil {
		return js.Item{}, err
	}

	return result, nil
}

/**
* Exec execute a command in the database
* @return js.Items
* @return error
**/
func (l *Linq) Exec() (js.Item, error) {
	if l.TypeQuery != TpCommand {
		return js.Item{}, logs.Alertm("The query is not a command")
	}

	c := l.Values
	switch c.TypeCommand {
	case TpInsert:
		err := c.Insert()
		if err != nil {
			return js.Item{}, err
		}
	case TpUpdate:
		err := c.Update()
		if err != nil {
			return js.Item{}, err
		}
	case TpDelete:
		err := c.Delete()
		if err != nil {
			return js.Item{}, err
		}
	}

	return *l.Result, nil
}

/**
* Go is a Exec function and return items
* @return js.Items
* @return error
**/
func (l *Linq) Go() (js.Item, error) {
	return l.Exec()
}

/**
* Query is a function to execute a query
* @return js.Items
* @return error
**/
func (l *Linq) Query() (js.Items, error) {
	var err error
	var items js.Items

	l.Sql, err = l.selectSql()
	if err != nil {
		return js.Items{}, err
	}

	if l.Datas.Used {
		items, err = l.data(l.Sql)
		if err != nil {
			return js.Items{}, err
		}
	} else {
		items, err = l.query(l.Sql)
		if err != nil {
			return js.Items{}, err
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
* @return js.Item
* @return error
**/
func (l *Linq) One() (js.Item, error) {
	items, err := l.Query()
	if err != nil {
		return js.Item{}, err
	}

	if items.Count == 0 {
		return js.Item{
			Ok:     false,
			Result: js.Json{},
		}, nil
	}

	return js.Item{
		Ok:     items.Ok,
		Result: items.Result[0],
	}, nil
}

/**
* Take is a function to select n element data
* @return js.Items
* @return error
**/
func (l *Linq) Take(n int) (js.Items, error) {
	l.Limit = n

	return l.Query()
}

/**
* Skip is a function to skip n element data
* @return js.Items
* @return error
**/
func (l *Linq) Skip(n int) (js.Items, error) {
	l.TypeQuery = TpSkip
	l.Limit = 1
	l.Offset = n

	return l.Query()
}

/**
* All is a function to select all data
* @return js.Items
* @return error
**/
func (l *Linq) All() (js.Items, error) {
	l.Limit = 0

	return l.Query()
}

/**
* Find is a function to select all data
* @return js.Items
* @return error
**/
func (l *Linq) Find() (js.Items, error) {
	return l.All()
}

/**
* First is a function to select first data
* @return js.Item
* @return error
**/
func (l *Linq) First() (js.Item, error) {
	items, err := l.Take(1)
	if err != nil {
		return js.Item{}, err
	}

	if !items.Ok {
		return js.Item{}, nil
	}

	return js.Item{
		Ok:     items.Ok,
		Result: items.Result[0],
	}, nil
}

/**
* Last is a function to select last data
* @return js.Item
* @return error
**/
func (l *Linq) Last() (js.Item, error) {
	l.TypeQuery = TpLast
	items, err := l.Take(1)
	if err != nil {
		return js.Item{}, err
	}

	if !items.Ok {
		return js.Item{}, nil
	}

	return js.Item{
		Ok:     items.Ok,
		Result: items.Result[0],
	}, nil
}

/**
* Page is a function to select page data
* @return js.Items
* @return error
**/
func (l *Linq) Page(page, rows int) (js.Items, error) {
	l.TypeQuery = TpPage
	offset := (page - 1) * rows
	l.Limit = rows
	l.Offset = offset

	return l.Query()
}

/**
* List is a function to select page data and return list
* @return js.List
* @return error
**/
func (l *Linq) List(page, rows int) (js.List, error) {
	l.TypeQuery = TpAll
	var err error
	l.Sql, err = l.selectSql()
	if err != nil {
		return js.List{}, err
	}

	item, err := l.One()
	if err != nil {
		return js.List{}, err
	}

	all := item.Int("count")

	items, err := l.Page(page, rows)
	if err != nil {
		return js.List{}, err
	}

	return items.ToList(all, page, rows), nil
}
