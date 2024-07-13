package linq

import (
	"database/sql"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
)

/**
* SQLQuote return a sql cuote string
* @param sql string
* @return string
**/
func SQLQuote(sql string) string {
	sql = strings.TrimSpace(sql)

	result := strs.Replace(sql, `'`, `"`)
	result = strs.Trim(result)

	return result
}

/**
* SQLDDL return a sql string with the args
* @param sql string
* @param args ...any
* @return string
**/
func SQLDDL(sql string, args ...any) string {
	sql = strings.TrimSpace(sql)

	for i, arg := range args {
		old := strs.Format(`$%d`, i+1)
		new := strs.Format(`%v`, arg)
		sql = strings.ReplaceAll(sql, old, new)
	}

	return sql
}

/**
* SQLParse return a sql string with the args
* @param sql string
* @param args ...any
* @return string
**/
func SQLParse(sql string, args ...any) string {
	for i := range args {
		old := strs.Format(`$%d`, i+1)
		new := strs.Format(`{$%d}`, i+1)
		sql = strings.ReplaceAll(sql, old, new)
	}

	for i, arg := range args {
		old := strs.Format(`{$%d}`, i+1)
		new := strs.Format(`%v`, et.Unquote(arg))
		sql = strings.ReplaceAll(sql, old, new)
	}

	return sql
}

/**
* RowsItems return a items from a sql rows
* @param rows *sql.Rows
* @return et.Items
**/
func RowsItems(rows *sql.Rows) et.Items {
	var result et.Items = et.Items{}

	for rows.Next() {
		var item et.Item
		err := item.Scan(rows)
		if err != nil {
			continue
		}

		result.Result = append(result.Result, item.Result)
		result.Ok = true
		result.Count++
	}

	return result
}

/**
* RowsItem return a item from a sql rows
* @param rows *sql.Rows
* @return et.Item
**/
func RowsItem(rows *sql.Rows) et.Item {
	items := RowsItems(rows)
	if items.Count == 0 {
		return et.Item{}
	}

	return et.Item{
		Ok:     items.Ok,
		Result: items.Result[0],
	}
}

/**
* DataItems return a items from a sql rows and source field
* @param rows *sql.Rows
* @param sourceField string
* @return et.Items
**/
func DataItems(rows *sql.Rows, sourceField string) et.Items {
	var result et.Items = et.Items{Result: []et.Json{}}

	for rows.Next() {
		var item et.Item
		err := item.Scan(rows)
		if err != nil {
			continue
		}

		result.Result = append(result.Result, item.Json(sourceField))
		result.Ok = true
		result.Count++
	}

	return result
}

/**
* DataItem return a item from a sql rows and source field
* @param rows *sql.Rows
* @param sourceField string
* @return et.Item
**/
func DataItem(rows *sql.Rows, sourceField string) et.Item {
	items := DataItems(rows, sourceField)
	if items.Count == 0 {
		return et.Item{}
	}

	return et.Item{
		Ok:     items.Ok,
		Result: items.Result[0],
	}
}
