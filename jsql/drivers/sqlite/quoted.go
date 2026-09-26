package sqlite

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
* Quoted: Formats an et.Value as the SQL literal that SQLite expects, chosen
* according to the value's declared Type (et.TEXT, et.KEY, et.MEMO, et.INT, et.DATETIME,
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
		return sqliteQuoteAggregate(v.Value)
	case et.VAL_NULL:
		return "NULL"
	case et.TEXT, et.KEY, et.MEMO:
		return sqliteQuoteString(v.Value)
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
		// SQLite has no native boolean type; 0/1 is the form its own INTEGER
		// casts (see sqliteAttribCast) round-trip correctly.
		if b {
			return "1"
		}
		return "0"
	case et.DATETIME:
		return sqliteQuoteTime(v.Value)
	case et.JSON:
		return sqliteQuoteJson(v.Value)
	case et.ARRAY_STRING, et.ARRAY_INT, et.ARRAY_FLOAT, et.ARRAY_BOOL, et.ARRAY_DATETIME, et.ARRAY_JSON:
		return sqliteQuoteArray(v)
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
* sqliteQuoteString: Quotes a string value as a SQLite string literal, escaping
* embedded single quotes.
* @param val any
* @return string
**/
func sqliteQuoteString(val any) string {
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
* sqliteQuoteTime: Quotes a datetime value as a SQLite text timestamp literal.
* @param val any
* @return string
**/
func sqliteQuoteTime(val any) string {
	switch t := val.(type) {
	case time.Time:
		return fmt.Sprintf("'%s'", t.Format("2006-01-02 15:04:05"))
	case *time.Time:
		if t == nil {
			return "NULL"
		}
		return fmt.Sprintf("'%s'", t.Format("2006-01-02 15:04:05"))
	case string:
		return sqliteQuoteString(t)
	case nil:
		return "NULL"
	default:
		logs.Errorf("Quoted, unexpected datetime value type:%v, value:%v", reflect.TypeOf(val), val)
		return "NULL"
	}
}

/**
* sqliteQuoteJson: Quotes a JSON value as a SQLite text literal holding the
* JSON payload. SQLite has no jsonb type or "::type" cast operator like
* Postgres — JSON columns are plain TEXT read back through json_extract/
* json_object/json_patch, so the value is just a quoted JSON string.
* @param val any
* @return string
**/
func sqliteQuoteJson(val any) string {
	switch j := val.(type) {
	case nil:
		return "NULL"
	case et.Json:
		return sqliteQuoteString(j.ToString())
	case map[string]interface{}:
		return sqliteQuoteString(et.Json(j).ToString())
	default:
		bt, err := json.Marshal(val)
		if err != nil {
			logs.Errorf("Quoted, error marshalling json value:%v, error:%v", val, err)
			return "NULL"
		}
		return sqliteQuoteString(string(bt))
	}
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
* sqliteQuoteArray: Formats an ARRAY_* et.Value as a parenthesized,
* comma-separated list of quoted elements, ready for use in an IN (...) clause.
* @param v et.Value
* @return string
**/
func sqliteQuoteArray(v et.Value) string {
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
* sqliteQuoteAggregate: Renders an et.Aggregate (value or pointer) as a SQL expression.
* A string operand is used verbatim (field reference or expression); any other operand is quoted.
* @param val any
* @return string
**/
func sqliteQuoteAggregate(val any) string {
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
		return fmt.Sprintf("CAST(strftime('%%Y', %s) AS INTEGER)", expr)
	case et.EXTRACT_MONTH:
		return fmt.Sprintf("CAST(strftime('%%m', %s) AS INTEGER)", expr)
	case et.EXTRACT_DAY:
		return fmt.Sprintf("CAST(strftime('%%d', %s) AS INTEGER)", expr)
	case et.EXTRACT_HOUR:
		return fmt.Sprintf("CAST(strftime('%%H', %s) AS INTEGER)", expr)
	case et.EXTRACT_MINUTE:
		return fmt.Sprintf("CAST(strftime('%%M', %s) AS INTEGER)", expr)
	case et.EXTRACT_SECOND:
		return fmt.Sprintf("CAST(strftime('%%S', %s) AS INTEGER)", expr)
	case et.INT_VALUE:
		return fmt.Sprintf("CAST(%s AS INTEGER)", expr)
	case et.FLOAT_VALUE:
		return fmt.Sprintf("CAST(%s AS REAL)", expr)
	default:
		return expr
	}
}
