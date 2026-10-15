package et

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

/**
* Str: The field reference as text, as ToField reads it: [function(][source.]name[)][:as]
* @return string
**/
func (s Field) Str() string {
	name := s.Name
	if s.Source != "" {
		name = s.Source + "." + name
	}
	if s.Agg != nil {
		fn := s.Agg.Function.Str()
		name = fmt.Sprintf("%s(%s)", fn, name)
		if s.As != "" && s.As != fn {
			name += ":" + s.As
		}
		return name
	}
	if s.As != "" && s.As != s.Name {
		name += ":" + s.As
	}
	return name
}

/**
* ValueOf: The value of the field in obj: the attribute of its path (name->a->b; the source is not used) with its
* function applied when it is an aggregate.
* @param obj Json
* @return any
**/
func (s Field) ValueOf(obj Json) any {
	val := pathValue(obj, s.Name)
	if s.Agg != nil {
		return applyFunction(s.Agg.Function, val)
	}
	return val
}

/**
* pathValue: The value of a path "a->b->c" in obj; nil when a key is missing. A list on the way gives the list of the
* rest of the path in each item ("items->price" over [{price: 2}, {price: 3}] is [2, 3]), for the aggregates.
* @param obj Json, path string
* @return any
**/
func pathValue(obj any, path string) any {
	key, rest, more := strings.Cut(path, "->")
	var val any
	switch v := obj.(type) {
	case Json:
		val = v[key]
	case map[string]any:
		val = v[key]
	default:
		if _, ok := obj.([]byte); ok || obj == nil {
			return nil
		}
		rv := reflect.ValueOf(obj)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return nil
		}
		result := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			if item := pathValue(rv.Index(i).Interface(), path); item != nil {
				result = append(result, item)
			}
		}
		return result
	}
	if !more {
		return val
	}
	return pathValue(val, rest)
}

/**
* fieldRef: Whether v is a field reference: a Field, or the JSON {"field": "reference"} (its only key, a text).
* A reference that ToField does not read is kept by name (Fld).
* @param v any
* @return Field, bool
**/
func fieldRef(v any) (Field, bool) {
	var obj map[string]any
	switch t := v.(type) {
	case Field:
		return t, true
	case *Field:
		if t == nil {
			return Field{}, false
		}
		return *t, true
	case Json:
		obj = t
	case map[string]any:
		obj = t
	default:
		return Field{}, false
	}
	if len(obj) != 1 {
		return Field{}, false
	}
	ref, ok := obj[FIELD.Str()].(string)
	if !ok || strings.TrimSpace(ref) == "" {
		return Field{}, false
	}
	return Fld(strings.TrimSpace(ref)), true
}

/**
* toConditionValue: The value of a JSON condition: a field reference {"field": "name"} is a FIELD value; anything
* else, the value as it is.
* @param v any
* @return Value
**/
func toConditionValue(v any) Value {
	if ref, ok := fieldRef(v); ok {
		return NewValue(ref)
	}
	return NewValue(v)
}

/**
* jsonValue: The value of a condition as JSON: a FIELD value is {"field": "reference"}.
* @param v Value
* @return any
**/
func jsonValue(v Value) any {
	if ref, ok := v.Value.(Field); ok && v.Type == FIELD {
		return Json{FIELD.Str(): ref.Str()}
	}
	return v.Value
}

/**
* resolve: The condition with its FIELD value read from obj; the condition itself when its value is not a FIELD.
* @param obj Json
* @return *Condition
**/
func (s *Condition) resolve(obj Json) *Condition {
	ref, ok := s.Value.Value.(Field)
	if s.Value.Type != FIELD || !ok {
		return s
	}
	tmp := *s
	tmp.Value = NewValue(ref.ValueOf(obj))
	return &tmp
}

/**
* listOf: The items of a list (any slice); a single value is a list of one and nil an empty list.
* @param val any
* @return []any
**/
func listOf(val any) []any {
	if val == nil {
		return []any{}
	}
	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []any{val}
	}
	if _, ok := val.([]byte); ok {
		return []any{val}
	}
	result := make([]any, rv.Len())
	for i := range result {
		result[i] = rv.Index(i).Interface()
	}
	return result
}

/**
* timeOf: The value as a time: a time.Time, *time.Time or a date text.
* @param val any
* @return time.Time, bool
**/
func timeOf(val any) (time.Time, bool) {
	switch v := val.(type) {
	case time.Time:
		return v, true
	case *time.Time:
		if v != nil {
			return *v, true
		}
	case string:
		return parseDate(v)
	}
	return time.Time{}, false
}

/**
* floatOf: The value as a number: any number or a numeric text.
* @param val any
* @return float64, bool
**/
func floatOf(val any) (float64, bool) {
	if f, _, ok := numberToFloat64(val); ok {
		return f, true
	}
	if s, ok := val.(string); ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

/**
* applyFunction: Applies an aggregate function to the value of one object: count, sum, avg, min and max over a list
* (a single value is a list of one; items that are not numbers are skipped), extract_* over a date, int_value and
* float_value as a cast. exp, and a function with no value to work on, give nil.
* @param fn Function, val any
* @return any
**/
func applyFunction(fn Function, val any) any {
	switch fn {
	case COUNT:
		return len(listOf(val))
	case SUM, AVG, MIN, MAX:
		numbers := make([]float64, 0)
		for _, item := range listOf(val) {
			if f, ok := floatOf(item); ok {
				numbers = append(numbers, f)
			}
		}
		if len(numbers) == 0 {
			if fn == SUM {
				return float64(0)
			}
			return nil
		}
		result := numbers[0]
		sum := 0.0
		for _, f := range numbers {
			sum += f
			if fn == MIN {
				result = math.Min(result, f)
			} else if fn == MAX {
				result = math.Max(result, f)
			}
		}
		switch fn {
		case SUM:
			return sum
		case AVG:
			return sum / float64(len(numbers))
		}
		return result
	case EXTRACT_YEAR, EXTRACT_MONTH, EXTRACT_DAY, EXTRACT_HOUR, EXTRACT_MINUTE, EXTRACT_SECOND:
		t, ok := timeOf(val)
		if !ok {
			return nil
		}
		switch fn {
		case EXTRACT_YEAR:
			return t.Year()
		case EXTRACT_MONTH:
			return int(t.Month())
		case EXTRACT_DAY:
			return t.Day()
		case EXTRACT_HOUR:
			return t.Hour()
		case EXTRACT_MINUTE:
			return t.Minute()
		}
		return t.Second()
	case INT_VALUE:
		if f, ok := floatOf(val); ok {
			return int64(f)
		}
		return nil
	case FLOAT_VALUE:
		if f, ok := floatOf(val); ok {
			return f
		}
		return nil
	}
	return nil
}
