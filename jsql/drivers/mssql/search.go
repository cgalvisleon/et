package mssql

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* msSearchExpr: Renders a search condition with LIKE (its full-text search needs a catalog, outside Load): each term of the
* text in one of the fields, the excluded in none, without index nor relevance (jsql.SearchLike). A field that does
* not resolve never matches.
* @param getField func(string) (*jsql.Field, bool), cond *et.Condition
* @return string
**/
func msSearchExpr(getField func(string) (*jsql.Field, bool), cond *et.Condition) string {
	fields, _, ok := jsql.SearchFields(getField, cond.Field.String())
	text, _, isText := jsql.SearchOf(cond) // without text search the lang does not change anything
	if !ok || !isText {
		return "1 = 0"
	}
	exprs := make([]string, 0, len(fields))
	for _, fld := range fields {
		exprs = append(exprs, msScalarExpr(fld, et.TEXT))
	}
	return jsql.SearchLike(exprs, text, func(expr, pattern string) string {
		return fmt.Sprintf("LOWER(%s) LIKE LOWER(%s) ESCAPE '\\'", expr, Quoted(et.Value{Type: et.TEXT, Value: pattern}))
	})
}
