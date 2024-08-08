package linq

import (
	"database/sql"
	"strings"

	"github.com/cgalvisleon/et/js"
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
		new := strs.Format(`%v`, js.Unquote(arg))
		sql = strings.ReplaceAll(sql, old, new)
	}

	return sql
}

/**
* RowsToItems return a items from a sql rows
* @param rows *sql.Rows
* @return js.Items
**/
func RowsToItems(rows *sql.Rows) js.Items {
	var result js.Items = js.Items{}
	for rows.Next() {
		var item js.Json
		item.ScanRows(rows)

		result.Ok = true
		result.Count++
		result.Result = append(result.Result, item)
	}

	return result
}

/**
* RowsToItem return a item from a sql rows
* @param rows *sql.Rows
* @return js.Item
**/
func RowsToItem(rows *sql.Rows) js.Item {
	var result js.Item = js.Item{}
	for rows.Next() {
		var item js.Json
		item.ScanRows(rows)

		result.Ok = true
		result.Result = item
		break
	}

	return result
}

/**
* DataItems return a items from a sql rows and source field
* @param rows *sql.Rows
* @param source string
* @return js.Items
**/
func DataToItems(rows *sql.Rows, source string) js.Items {
	var result js.Items = js.Items{}
	for rows.Next() {
		var item js.Json
		item.ScanRows(rows)

		result.Ok = true
		result.Count++
		result.Result = append(result.Result, item.Json(source))
	}

	return result
}

/**
* DataItem return a item from a sql rows and source field
* @param rows *sql.Rows
* @param source string
* @return js.Item
**/
func DataToItem(rows *sql.Rows, source string) js.Item {
	var result js.Item = js.Item{}
	for rows.Next() {
		var item js.Json
		item.ScanRows(rows)

		result.Ok = true
		result.Result = item.Json(source)
		break
	}

	return result
}
