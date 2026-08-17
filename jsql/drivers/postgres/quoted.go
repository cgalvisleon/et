package postgres

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
)

/**
* Quoted: Formats an et.Value as the SQL literal that PostgreSQL expects,
* chosen according to the value's declared Type (et.STRING, et.INT, et.DATETIME,
* et.JSON, the ARRAY_* types, et.VAL_BETWEEN, et.VAL_NULL, etc.).
* et.EXPR is the one type returned verbatim, with no quoting at all: it carries
* a raw SQL fragment rather than a literal value, e.g. COUNT(*) or a field
* reference from another table required by a JOIN or expression.
* @param v et.Value
* @return string
**/
func Quoted(v et.Value) string {
	switch v.Type {
	case et.EXPR:
		s, _ := v.Value.(string)
		return s
	case et.VAL_NULL:
		return "NULL"
	case et.STRING:
		return pgQuoteString(v.Value)
	case et.INT, et.FLOAT:
		if v.Value == nil {
			return "NULL"
		}
		return fmt.Sprintf("%v", v.Value)
	case et.BOOL:
		b, ok := v.Value.(bool)
		if !ok {
			return "NULL"
		}
		return fmt.Sprintf("%v", b)
	case et.DATETIME:
		return pgQuoteTime(v.Value)
	case et.JSON:
		return pgQuoteJson(v.Value)
	case et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME, et.ARRAY_JSON:
		return pgQuoteArray(v)
	case et.VAL_BETWEEN:
		bv, ok := v.Value.(et.BetweenValue)
		if !ok {
			return "NULL"
		}
		return fmt.Sprintf("%s AND %s", Quoted(et.NewValue(bv.Min)), Quoted(et.NewValue(bv.Max)))
	default: // et.ANY and any unrecognized type: infer quoting from the raw Go value
		return fmt.Sprintf("%v", jsql.Quoted(v.Value))
	}
}

/**
* pgQuoteString: Quotes a string value as a PostgreSQL string literal, escaping
* embedded single quotes.
* @param val any
* @return string
**/
func pgQuoteString(val any) string {
	if val == nil {
		return "NULL"
	}
	s, ok := val.(string)
	if !ok {
		s = fmt.Sprintf("%v", val)
	}
	return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "''"))
}

/**
* pgQuoteTime: Quotes a datetime value as a PostgreSQL timestamp literal.
* @param val any
* @return string
**/
func pgQuoteTime(val any) string {
	switch t := val.(type) {
	case time.Time:
		return fmt.Sprintf("'%s'", t.Format("2006-01-02 15:04:05"))
	case *time.Time:
		if t == nil {
			return "NULL"
		}
		return fmt.Sprintf("'%s'", t.Format("2006-01-02 15:04:05"))
	case string:
		return pgQuoteString(t)
	case nil:
		return "NULL"
	default:
		logs.Errorf("Quoted, unexpected datetime value type:%v, value:%v", reflect.TypeOf(val), val)
		return "NULL"
	}
}

/**
* pgQuoteJson: Quotes a JSON value as a PostgreSQL jsonb literal.
* @param val any
* @return string
**/
func pgQuoteJson(val any) string {
	switch j := val.(type) {
	case nil:
		return "NULL"
	case et.Json:
		return fmt.Sprintf("'%s'::jsonb", strings.ReplaceAll(j.ToString(), "'", "''"))
	case map[string]interface{}:
		return fmt.Sprintf("'%s'::jsonb", strings.ReplaceAll(et.Json(j).ToString(), "'", "''"))
	default:
		bt, err := json.Marshal(val)
		if err != nil {
			logs.Errorf("Quoted, error marshalling json value:%v, error:%v", val, err)
			return "NULL"
		}
		return fmt.Sprintf("'%s'::jsonb", strings.ReplaceAll(string(bt), "'", "''"))
	}
}

/**
* arrayElemType: Maps an ARRAY_* et.Value type to the logical type of its elements.
* @param tp string
* @return string
**/
func arrayElemType(tp string) string {
	switch tp {
	case et.ARRAY_STRING:
		return et.STRING
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
* pgQuoteArray: Formats an ARRAY_* et.Value as a parenthesized, comma-separated
* list of quoted elements, ready for use in an IN (...) clause.
* @param v et.Value
* @return string
**/
func pgQuoteArray(v et.Value) string {
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
