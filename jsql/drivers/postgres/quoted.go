package postgres

import (
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
* chosen according to the value's declared Type (et.TEXT, et.KEY, et.MEMO, et.INT, et.DATETIME,
* et.JSON, the ARRAY_* types, et.VAL_BETWEEN, et.VAL_NULL, etc.).
* et.AGGREGATE is not quoted: it carries an et.Aggregate whose operand is a raw SQL
* fragment (a field reference or expression), rendered as the function applied to it,
* e.g. COUNT(A.id); et.EXP returns the operand verbatim.
* @param v et.Value
* @return string
**/
func Quoted(v et.Value) string {
	switch v.Type {
	case et.AGGREGATE:
		return pgQuoteAggregate(v.Value)
	case et.VAL_NULL:
		return "NULL"
	case et.TEXT, et.KEY, et.MEMO:
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
	if val == nil {
		return "NULL"
	}
	str, err := jsql.JsonString(val)
	if err != nil {
		logs.Errorf("Quoted, error marshalling json value:%v, error:%v", val, err)
		return "NULL"
	}
	return fmt.Sprintf("'%s'::jsonb", jsql.EscapeSQLString(str))
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

/**
* pgQuoteAggregate: Renders an et.Aggregate (value or pointer) as a SQL expression.
* A string operand is used verbatim (field reference or expression); any other operand is quoted.
* @param val any
* @return string
**/
func pgQuoteAggregate(val any) string {
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

	switch agg.Function {
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
		return fmt.Sprintf("(%s)::bigint", expr)
	case et.FLOAT_VALUE:
		return fmt.Sprintf("(%s)::double precision", expr)
	default:
		return expr
	}
}
