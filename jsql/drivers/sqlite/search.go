package sqlite

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* sqliteSearchExpr: Renders a search condition with LIKE (its FTS5 needs virtual tables and triggers, outside Load): each term of the
* text in one of the fields, the excluded in none, without index nor relevance (jsql.SearchLike). A field that does
* not resolve never matches.
* @param getField func(string) (*jsql.Field, bool), useSourceField bool, cond *et.Condition
* @return string
**/
func sqliteSearchExpr(getField func(string) (*jsql.Field, bool), useSourceField bool, cond *et.Condition) string {
	fields, _, ok := jsql.SearchFields(getField, cond.Field.String())
	text, _, isText := jsql.SearchOf(cond) // without text search the lang does not change anything
	if !ok || !isText {
		return "1 = 0"
	}
	exprs := make([]string, 0, len(fields))
	for _, fld := range fields {
		exprs = append(exprs, sqliteFieldExpr(fld, useSourceField))
	}
	return jsql.SearchLike(exprs, text, func(expr, pattern string) string {
		return fmt.Sprintf("%s LIKE %s ESCAPE '\\'", expr, Quoted(et.Value{Type: et.TEXT, Value: pattern}))
	})
}
