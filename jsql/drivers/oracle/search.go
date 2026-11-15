package oracle

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* oraSearchExpr: Renders a search condition with LIKE (Oracle Text needs its own indexes and grants, outside Load): each term of the
* text in one of the fields, the excluded in none, without index nor relevance (jsql.SearchLike). A field that does
* not resolve never matches.
* @param getField func(string) (*jsql.Field, bool), cond *et.Condition
* @return string
**/
func oraSearchExpr(getField func(string) (*jsql.Field, bool), cond *et.Condition) string {
	fields, _, ok := jsql.SearchFields(getField, cond.Field.String())
	text, _, isText := jsql.SearchOf(cond) // without text search the lang does not change anything
	if !ok || !isText {
		return "1 = 0"
	}
	exprs := make([]string, 0, len(fields))
	for _, fld := range fields {
		exprs = append(exprs, oraScalarExpr(fld, et.TEXT))
	}
	return jsql.SearchLike(exprs, text, func(expr, pattern string) string {
		return fmt.Sprintf("UPPER(%s) LIKE UPPER(%s) ESCAPE '\\'", expr, Quoted(et.Value{Type: et.TEXT, Value: pattern}))
	})
}
