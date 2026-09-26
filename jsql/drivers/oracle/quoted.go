package oracle

import (
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
)

// maxLiteralChars keeps each string literal under the 4000-byte SQL limit
// (ORA-01704) even when every character takes 4 bytes in UTF-8.
const maxLiteralChars = 1000

// timeLayouts are the string formats accepted for DATETIME values; the ISO form without zone
// is how JSON_OBJECT returns a TIMESTAMP.
var timeLayouts = []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05", "2006-01-02"}

/**
* Quoted: Formats an et.Value as the SQL literal that Oracle expects, chosen according to
* the value's declared Type. et.AGGREGATE is rendered as the function applied to its operand.
* @param v et.Value
* @return string
**/
func Quoted(v et.Value) string {
	switch v.Type {
	case et.AGGREGATE:
		return oraQuoteAggregate(v.Value)
	case et.VAL_NULL:
		return "NULL"
	case et.TEXT, et.KEY, et.MEMO:
		return oraQuoteString(v.Value)
	case et.INT, et.FLOAT:
		if v.Value == nil {
			return "NULL"
		}
		return fmt.Sprintf("%v", v.Value)
	case et.BOOL:
		return oraQuoteBool(v.Value)
	case et.DATETIME:
		return oraQuoteTime(v.Value)
	case et.JSON:
		return oraQuoteJson(v.Value)
	case et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME, et.ARRAY_JSON:
		return oraQuoteArray(v)
	case et.VAL_BETWEEN:
		bv, ok := v.Value.(et.BetweenValue)
		if !ok {
			return "NULL"
		}
		return fmt.Sprintf("%s AND %s", Quoted(et.NewValue(bv.Min)), Quoted(et.NewValue(bv.Max)))
	default:
		return oraQuoteAny(v.Value)
	}
}

/**
* QuotedAs: Formats val as a literal for a column of type tp (used for INSERT/UPDATE values),
* so booleans become 1/0, times TO_TIMESTAMP(...) and JSON a CLOB literal.
* @param tp et.TypeData, val any
* @return string
**/
func QuotedAs(tp et.TypeData, val any) string {
	if val == nil {
		return "NULL"
	}
	switch tp {
	case et.TEXT, et.KEY, et.MEMO:
		return oraQuoteString(val)
	case et.INT, et.FLOAT:
		switch val.(type) {
		case string:
			return oraQuoteString(val)
		case bool:
			return oraQuoteBool(val)
		}
		return fmt.Sprintf("%v", val)
	case et.BOOL:
		return oraQuoteBool(val)
	case et.DATETIME:
		return oraQuoteTime(val)
	case et.JSON, et.ARRAY, et.ARRAY_JSON, et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME, et.VAL_BETWEEN:
		return oraQuoteJson(val)
	default:
		return Quoted(et.NewValue(val))
	}
}

/**
* oraQuoteAny: Infers the literal from the Go type of val.
* @param val any
* @return string
**/
func oraQuoteAny(val any) string {
	switch v := val.(type) {
	case nil:
		return "NULL"
	case string:
		return oraQuoteString(v)
	case bool:
		return oraQuoteBool(v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", v)
	case time.Time, *time.Time:
		return oraQuoteTime(v)
	case et.Json, map[string]any, []any, []et.Json, []map[string]any, []string:
		return oraQuoteJson(v)
	case []byte:
		return fmt.Sprintf("HEXTORAW('%x')", v)
	default:
		logs.Errorf("Oracle quoted, type:%v, value:%v", reflect.TypeOf(val), val)
		return oraQuoteString(fmt.Sprintf("%v", v))
	}
}

/**
* oraQuoteString: Quotes a value as an Oracle string literal, escaping single quotes.
* Values longer than the 4000-byte literal limit become TO_CLOB('…') || TO_CLOB('…').
* @param val any
* @return string
**/
func oraQuoteString(val any) string {
	if val == nil {
		return "NULL"
	}
	s, ok := val.(string)
	if !ok {
		s = fmt.Sprintf("%v", val)
	}
	if utf8.RuneCountInString(s) <= maxLiteralChars {
		return fmt.Sprintf("'%s'", jsql.EscapeSQLString(s))
	}
	return oraClob(s)
}

/**
* oraClob: Returns s as a CLOB expression built from literals of at most maxLiteralChars characters.
* @param s string
* @return string
**/
func oraClob(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return "TO_CLOB('')"
	}
	parts := make([]string, 0, len(runes)/maxLiteralChars+1)
	for i := 0; i < len(runes); i += maxLiteralChars {
		end := min(i+maxLiteralChars, len(runes))
		parts = append(parts, fmt.Sprintf("TO_CLOB('%s')", jsql.EscapeSQLString(string(runes[i:end]))))
	}
	return strings.Join(parts, " || ")
}

/**
* oraQuoteBool: Formats a boolean as 1/0 (Oracle 19c has no SQL boolean type).
* @param val any
* @return string
**/
func oraQuoteBool(val any) string {
	switch v := val.(type) {
	case bool:
		if v {
			return "1"
		}
		return "0"
	case string:
		switch strings.ToLower(v) {
		case "true", "1":
			return "1"
		case "false", "0":
			return "0"
		}
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	}
	return "NULL"
}

/**
* oraQuoteTime: Formats a time as TO_TIMESTAMP('…', 'YYYY-MM-DD HH24:MI:SS.FF').
* Strings in RFC 3339 or "YYYY-MM-DD[ HH:MM:SS]" form are parsed first.
* @param val any
* @return string
**/
func oraQuoteTime(val any) string {
	format := func(t time.Time) string {
		return fmt.Sprintf("TO_TIMESTAMP('%s', 'YYYY-MM-DD HH24:MI:SS.FF')", t.Format("2006-01-02 15:04:05.000000"))
	}
	switch t := val.(type) {
	case nil:
		return "NULL"
	case time.Time:
		return format(t)
	case *time.Time:
		if t == nil {
			return "NULL"
		}
		return format(*t)
	case string:
		for _, layout := range timeLayouts {
			if parsed, err := time.Parse(layout, t); err == nil {
				return format(parsed)
			}
		}
		return oraQuoteString(t)
	default:
		logs.Errorf("Oracle quoted, unexpected datetime value type:%v, value:%v", reflect.TypeOf(val), val)
		return "NULL"
	}
}

/**
* oraQuoteJson: Serializes val as JSON and returns it as a CLOB literal.
* @param val any
* @return string
**/
func oraQuoteJson(val any) string {
	if val == nil {
		return "NULL"
	}
	str, err := jsql.JsonString(val)
	if err != nil {
		logs.Errorf("Oracle quoted, error marshalling json value:%v, error:%v", val, err)
		return "NULL"
	}
	return oraClob(str)
}

/**
* arrayElemType: Maps an ARRAY_* et.Value type to the logical type of its elements.
* @param tp et.TypeData
* @return et.TypeData
**/
func arrayElemType(tp et.TypeData) et.TypeData {
	switch tp {
	case et.ARRAY_STRING:
		return et.TEXT
	case et.ARRAY_INT:
		return et.INT
	case et.ARRAY_FLOAT:
		return et.FLOAT
	case et.ARRAY_BOOL:
		return et.BOOL
	case et.ARRAY_DATETIME:
		return et.DATETIME
	case et.ARRAY_JSON:
		return et.JSON
	default:
		return et.ANY
	}
}

/**
* oraQuoteArray: Formats an ARRAY_* et.Value as a parenthesized list of quoted elements (IN lists).
* @param v et.Value
* @return string
**/
func oraQuoteArray(v et.Value) string {
	items := reflect.ValueOf(v.Value)
	if !items.IsValid() || (items.Kind() != reflect.Slice && items.Kind() != reflect.Array) {
		return "NULL"
	}

	elemType := arrayElemType(v.Type)
	parts := make([]string, items.Len())
	for i := 0; i < items.Len(); i++ {
		parts[i] = Quoted(et.Value{Type: elemType, Value: items.Index(i).Interface()})
	}
	return fmt.Sprintf("(%s)", strings.Join(parts, ", "))
}

/**
* oraQuoteAggregate: Renders an et.Aggregate (value or pointer) as a SQL expression.
* A string operand is used verbatim (field reference or expression); any other operand is quoted.
* @param val any
* @return string
**/
func oraQuoteAggregate(val any) string {
	var agg et.Aggregate
	switch t := val.(type) {
	case et.Aggregate:
		agg = t
	case *et.Aggregate:
		if t == nil {
			return "NULL"
		}
		agg = *t
	default:
		return "NULL"
	}

	expr, ok := agg.Value.Value.(string)
	if !ok {
		expr = Quoted(agg.Value)
	}
	return oraAggregate(agg.Function, expr)
}

/**
* oraAggregate: Applies an aggregate or scalar function to a SQL expression.
* @param fn et.Function, expr string
* @return string
**/
func oraAggregate(fn et.Function, expr string) string {
	switch fn {
	case et.COUNT:
		return fmt.Sprintf("COUNT(%s)", expr)
	case et.SUM:
		return fmt.Sprintf("SUM(%s)", expr)
	case et.AVG:
		return fmt.Sprintf("AVG(%s)", expr)
	case et.MIN:
		return fmt.Sprintf("MIN(%s)", expr)
	case et.MAX:
		return fmt.Sprintf("MAX(%s)", expr)
	case et.EXTRACT_YEAR:
		return fmt.Sprintf("EXTRACT(YEAR FROM %s)", expr)
	case et.EXTRACT_MONTH:
		return fmt.Sprintf("EXTRACT(MONTH FROM %s)", expr)
	case et.EXTRACT_DAY:
		return fmt.Sprintf("EXTRACT(DAY FROM %s)", expr)
	case et.EXTRACT_HOUR:
		return fmt.Sprintf("EXTRACT(HOUR FROM %s)", expr)
	case et.EXTRACT_MINUTE:
		return fmt.Sprintf("EXTRACT(MINUTE FROM %s)", expr)
	case et.EXTRACT_SECOND:
		return fmt.Sprintf("EXTRACT(SECOND FROM %s)", expr)
	case et.INT_VALUE:
		return fmt.Sprintf("CAST(%s AS NUMBER(19))", expr)
	case et.FLOAT_VALUE:
		return fmt.Sprintf("CAST(%s AS NUMBER)", expr)
	default:
		return expr
	}
}
