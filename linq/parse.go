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
* RowsItems return a items from a sql rows
* @param rows *sql.Rows
* @return js.Items
**/
func RowsItems(rows *sql.Rows) js.Items {
	var result js.Items = js.Items{Result: []js.Json{}}
	for rows.Next() {
		var item js.Item
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
* @return js.Item
**/
func RowsItem(rows *sql.Rows) js.Item {
	var result js.Item = js.Item{Result: js.Json{}}
	for rows.Next() {
		err := result.Scan(rows)
		if err != nil {
			continue
		}

		break
	}

	return result
}

/**
* DataItems return a items from a sql rows and source field
* @param rows *sql.Rows
* @param sourceField string
* @return js.Items
**/
func DataItems(rows *sql.Rows, sourceField string) js.Items {
	var result js.Items = js.Items{Result: []js.Json{}}
	for rows.Next() {
		var item js.Item
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
* @return js.Item
**/
func DataItem(rows *sql.Rows, sourceField string) js.Item {
	var result js.Item = js.Item{Result: js.Json{}}
	for rows.Next() {
		err := result.Scan(rows)
		if err != nil {
			continue
		}

		break
	}

	result.Result = result.Json(sourceField)

	return result
}
