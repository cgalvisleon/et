package linq

import (
	"database/sql"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
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
		new := strs.Format(`%v`, utility.Unquote(arg))
		sql = strings.ReplaceAll(sql, old, new)
	}

	return sql
}

/**
* RowsToItems return a items from a sql rows
* @param rows *sql.Rows
* @return et.Items
**/
func RowsToItems(rows *sql.Rows) et.Items {
	var result et.Items = et.Items{}
	for rows.Next() {
		var item et.Json
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
* @return et.Item
**/
func RowsToItem(rows *sql.Rows) et.Item {
	var result et.Item = et.Item{}
	for rows.Next() {
		var item et.Json
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
* @return et.Items
**/
func DataToItems(rows *sql.Rows, source string) et.Items {
	var result et.Items = et.Items{}
	for rows.Next() {
		var item et.Json
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
* @return et.Item
**/
func DataToItem(rows *sql.Rows, source string) et.Item {
	var result et.Item = et.Item{}
	for rows.Next() {
		var item et.Json
		item.ScanRows(rows)

		result.Ok = true
		result.Result = item.Json(source)
		break
	}

	return result
}
