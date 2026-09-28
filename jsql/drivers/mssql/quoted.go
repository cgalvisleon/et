package mssql

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
)

// timeLayouts are the string formats accepted for DATETIME values; the ISO form without zone is how the
// driver returns a DATETIME2 in the row JSON.
var timeLayouts = []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05", "2006-01-02"}

/**
* msQuoteText: Quotes s as an N'…' string literal, escaping single quotes.
* @param s string
* @return string
**/
func msQuoteText(s string) string {
	return "N'" + strings.ReplaceAll(s, "'", "''") + "'"
}

/**
* Quoted: Formats an et.Value as the SQL literal SQL Server expects, chosen by the value's declared Type.
* @param v et.Value
* @return string
**/
func Quoted(v et.Value) string {
	switch v.Type {
	case et.AGGREGATE:
		return msQuoteAggregate(v.Value)
	case et.VAL_NULL:
		return "NULL"
	case et.TEXT, et.KEY, et.MEMO:
		return msQuoteString(v.Value)
	case et.INT, et.FLOAT:
		if v.Value == nil {
			return "NULL"
		}
		return fmt.Sprintf("%v", v.Value)
	case et.BOOL:
		return msQuoteBool(v.Value)
	case et.DATETIME:
		return msQuoteTime(v.Value)
	case et.JSON:
		return msQuoteJson(v.Value)
	case et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME, et.ARRAY_JSON:
		return msQuoteArray(v)
	case et.VAL_BETWEEN:
		bv, ok := v.Value.(et.BetweenValue)
		if !ok {
			return "NULL"
		}
		return fmt.Sprintf("%s AND %s", Quoted(et.NewValue(bv.Min)), Quoted(et.NewValue(bv.Max)))
	default:
		return msQuoteAny(v.Value)
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
		return msQuoteString(val)
	case et.INT, et.FLOAT:
		switch val.(type) {
		case string:
			return msQuoteString(val)
		case bool:
			return msQuoteBool(val)
		}
		return fmt.Sprintf("%v", val)
	case et.BOOL:
		return msQuoteBool(val)
	case et.DATETIME:
		return msQuoteTime(val)
	case et.JSON, et.ARRAY, et.ARRAY_JSON, et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME, et.VAL_BETWEEN:
		return msQuoteJson(val)
	default:
		return Quoted(et.NewValue(val))
	}
}

/**
* msQuoteAny: Infers the literal from the Go type of val.
* @param val any
* @return string
**/
func msQuoteAny(val any) string {
	switch v := val.(type) {
	case nil:
		return "NULL"
	case string:
		return msQuoteText(v)
	case bool:
		return msQuoteBool(v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", v)
	case time.Time, *time.Time:
		return msQuoteTime(v)
	case et.Json, map[string]any, []any, []et.Json, []map[string]any, []string:
		return msQuoteJson(v)
	case []byte:
		return fmt.Sprintf("0x%x", v)
	default:
		logs.Errorf("SQL Server quoted, type:%v, value:%v", reflect.TypeOf(val), val)
		return msQuoteText(fmt.Sprintf("%v", v))
	}
}

/**
* msQuoteString: Quotes a value as an N'…' literal.
* @param val any
* @return string
**/
func msQuoteString(val any) string {
	if val == nil {
		return "NULL"
	}
	s, ok := val.(string)
	if !ok {
		s = fmt.Sprintf("%v", val)
	}
	return msQuoteText(s)
}

/**
* msQuoteBool: Formats a boolean as 1/0 (BIT).
* @param val any
* @return string
**/
func msQuoteBool(val any) string {
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
* msQuoteTime: Formats a time as the language-neutral ISO literal 'YYYY-MM-DDTHH:MM:SS.fffffff';
* strings are parsed first.
* @param val any
* @return string
**/
func msQuoteTime(val any) string {
	format := func(t time.Time) string {
		return fmt.Sprintf("'%s'", t.Format("2006-01-02T15:04:05.0000000"))
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
		return msQuoteText(t)
	default:
		logs.Errorf("SQL Server quoted, unexpected datetime value type:%v, value:%v", reflect.TypeOf(val), val)
		return "NULL"
	}
}

/**
* msQuoteJson: Serializes val as JSON and returns it as an N'…' literal.
* @param val any
* @return string
**/
func msQuoteJson(val any) string {
	if val == nil {
		return "NULL"
	}
	str, err := jsql.JsonString(val)
	if err != nil {
		logs.Errorf("SQL Server quoted, error marshalling json value:%v, error:%v", val, err)
		return "NULL"
	}
	return msQuoteText(str)
}

/**
* msModifyValue: Returns val as the value argument of JSON_MODIFY: objects and arrays through JSON_QUERY,
* booleans as BIT, numbers and strings as literals. A nil value makes JSON_MODIFY remove the key.
* @param val any
* @return string
**/
func msModifyValue(val any) string {
	switch v := val.(type) {
	case nil:
		return "NULL"
	case bool:
		return fmt.Sprintf("CAST(%s AS BIT)", msQuoteBool(v))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", v)
	case string:
		return msQuoteText(v)
	case time.Time:
		return msQuoteText(v.Format(time.RFC3339Nano))
	case *time.Time:
		if v == nil {
			return "NULL"
		}
		return msQuoteText(v.Format(time.RFC3339Nano))
	default:
		return fmt.Sprintf("JSON_QUERY(%s)", msQuoteJson(v))
	}
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
* msQuoteArray: Formats an ARRAY_* value as a parenthesized list (IN lists).
* @param v et.Value
* @return string
**/
func msQuoteArray(v et.Value) string {
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
* msQuoteAggregate: Renders an et.Aggregate as a SQL expression.
* @param val any
* @return string
**/
func msQuoteAggregate(val any) string {
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
	return msAggregate(agg.Function, expr)
}

/**
* msAggregate: Applies an aggregate or scalar function to a SQL expression.
* @param fn et.Function, expr string
* @return string
**/
func msAggregate(fn et.Function, expr string) string {
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
		return fmt.Sprintf("DATEPART(YEAR, %s)", expr)
	case et.EXTRACT_MONTH:
		return fmt.Sprintf("DATEPART(MONTH, %s)", expr)
	case et.EXTRACT_DAY:
		return fmt.Sprintf("DATEPART(DAY, %s)", expr)
	case et.EXTRACT_HOUR:
		return fmt.Sprintf("DATEPART(HOUR, %s)", expr)
	case et.EXTRACT_MINUTE:
		return fmt.Sprintf("DATEPART(MINUTE, %s)", expr)
	case et.EXTRACT_SECOND:
		return fmt.Sprintf("DATEPART(SECOND, %s)", expr)
	case et.INT_VALUE:
		return fmt.Sprintf("CAST(%s AS BIGINT)", expr)
	case et.FLOAT_VALUE:
		return fmt.Sprintf("CAST(%s AS DECIMAL(38,10))", expr)
	default:
		return expr
	}
}
