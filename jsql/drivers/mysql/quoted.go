package mysql

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
)

// timeLayouts are the string formats accepted for DATETIME values; the last ones are how
// JSON_OBJECT returns a DATETIME.
var timeLayouts = []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05", "2006-01-02"}

/**
* myQuoteText: Quotes s as a MySQL string literal. MySQL treats the backslash as an escape character
* inside strings, so both \ and ' are escaped.
* @param s string
* @return string
**/
func myQuoteText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `''`)
	return "'" + s + "'"
}

/**
* Quoted: Formats an et.Value as the SQL literal MySQL expects, chosen by the value's declared Type.
* @param v et.Value
* @return string
**/
func Quoted(v et.Value) string {
	switch v.Type {
	case et.AGGREGATE:
		return myQuoteAggregate(v.Value)
	case et.VAL_NULL:
		return "NULL"
	case et.TEXT, et.KEY, et.MEMO:
		return myQuoteString(v.Value)
	case et.INT, et.FLOAT:
		if v.Value == nil {
			return "NULL"
		}
		return fmt.Sprintf("%v", v.Value)
	case et.BOOL:
		return myQuoteBool(v.Value)
	case et.DATETIME:
		return myQuoteTime(v.Value)
	case et.JSON:
		return myQuoteJson(v.Value)
	case et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME, et.ARRAY_JSON:
		return myQuoteArray(v)
	case et.VAL_BETWEEN:
		bv, ok := v.Value.(et.BetweenValue)
		if !ok {
			return "NULL"
		}
		return fmt.Sprintf("%s AND %s", Quoted(et.NewValue(bv.Min)), Quoted(et.NewValue(bv.Max)))
	default:
		return myQuoteAny(v.Value)
	}
}

/**
* QuotedAs: Formats val as a literal for a column of type tp (INSERT/UPDATE values).
* @param tp et.TypeData, val any
* @return string
**/
func QuotedAs(tp et.TypeData, val any) string {
	if val == nil {
		return "NULL"
	}
	switch tp {
	case et.TEXT, et.KEY, et.MEMO:
		return myQuoteString(val)
	case et.INT, et.FLOAT:
		switch val.(type) {
		case string:
			return myQuoteString(val)
		case bool:
			return myQuoteBool(val)
		}
		return fmt.Sprintf("%v", val)
	case et.BOOL:
		return myQuoteBool(val)
	case et.DATETIME:
		return myQuoteTime(val)
	case et.JSON, et.ARRAY, et.ARRAY_JSON, et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME, et.VAL_BETWEEN:
		return myQuoteJson(val)
	default:
		return Quoted(et.NewValue(val))
	}
}

/**
* myQuoteAny: Infers the literal from the Go type of val.
* @param val any
* @return string
**/
func myQuoteAny(val any) string {
	switch v := val.(type) {
	case nil:
		return "NULL"
	case string:
		return myQuoteText(v)
	case bool:
		return myQuoteBool(v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", v)
	case time.Time, *time.Time:
		return myQuoteTime(v)
	case et.Json, map[string]any, []any, []et.Json, []map[string]any, []string:
		return myQuoteJson(v)
	case []byte:
		return fmt.Sprintf("X'%x'", v)
	default:
		logs.Errorf("MySQL quoted, type:%v, value:%v", reflect.TypeOf(val), val)
		return myQuoteText(fmt.Sprintf("%v", v))
	}
}

/**
* myQuoteString: Quotes a value as a string literal.
* @param val any
* @return string
**/
func myQuoteString(val any) string {
	if val == nil {
		return "NULL"
	}
	s, ok := val.(string)
	if !ok {
		s = fmt.Sprintf("%v", val)
	}
	return myQuoteText(s)
}

/**
* myQuoteBool: Formats a boolean as TRUE/FALSE.
* @param val any
* @return string
**/
func myQuoteBool(val any) string {
	switch v := val.(type) {
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	case string:
		switch strings.ToLower(v) {
		case "true", "1":
			return "TRUE"
		case "false", "0":
			return "FALSE"
		}
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	}
	return "NULL"
}

/**
* myQuoteTime: Formats a time as 'YYYY-MM-DD HH:MM:SS.ffffff'; strings are parsed first.
* @param val any
* @return string
**/
func myQuoteTime(val any) string {
	format := func(t time.Time) string {
		return fmt.Sprintf("'%s'", t.Format("2006-01-02 15:04:05.000000"))
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
		return myQuoteText(t)
	default:
		logs.Errorf("MySQL quoted, unexpected datetime value type:%v, value:%v", reflect.TypeOf(val), val)
		return "NULL"
	}
}

/**
* myQuoteJson: Serializes val as JSON and returns it as a string literal (accepted by JSON columns).
* @param val any
* @return string
**/
func myQuoteJson(val any) string {
	if val == nil {
		return "NULL"
	}
	str, err := jsql.JsonString(val)
	if err != nil {
		logs.Errorf("MySQL quoted, error marshalling json value:%v, error:%v", val, err)
		return "NULL"
	}
	return myQuoteText(str)
}

/**
* myJsonValue: Returns val as a JSON value (CAST('…' AS JSON)) for JSON_SET.
* @param val any
* @return string
**/
func myJsonValue(val any) string {
	str, err := jsql.JsonString(val)
	if err != nil {
		return "CAST('null' AS JSON)"
	}
	return fmt.Sprintf("CAST(%s AS JSON)", myQuoteText(str))
}

/**
* arrayElemType: Maps an ARRAY_* type to the type of its elements.
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
* myQuoteArray: Formats an ARRAY_* value as a parenthesized list (IN lists).
* @param v et.Value
* @return string
**/
func myQuoteArray(v et.Value) string {
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
* myQuoteAggregate: Renders an et.Aggregate as a SQL expression.
* @param val any
* @return string
**/
func myQuoteAggregate(val any) string {
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
	return myAggregate(agg.Function, expr)
}

/**
* myAggregate: Applies an aggregate or scalar function to a SQL expression.
* @param fn et.Function, expr string
* @return string
**/
func myAggregate(fn et.Function, expr string) string {
	switch fn {
	case et.COUNT:
		return fmt.Sprintf("COUNT(%s)", expr)
	case et.SUM:
		return fmt.Sprintf("COALESCE(SUM(%s), 0)", expr)
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
		return fmt.Sprintf("CAST(%s AS SIGNED)", expr)
	case et.FLOAT_VALUE:
		return fmt.Sprintf("CAST(%s AS DECIMAL(38,10))", expr)
	default:
		return expr
	}
}
