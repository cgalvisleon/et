package sql

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
)

/**
* numberToFloat64: Converts a number to float64
* @param v any
* @return float64, reflect.Kind, bool
**/
func numberToFloat64(v any) (float64, reflect.Kind, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return 0, reflect.Invalid, false
	}

	// Si llega un puntero, opcionalmente lo resolvemos
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return 0, reflect.Invalid, false
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), rv.Kind(), true

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(rv.Uint()), rv.Kind(), true

	case reflect.Float32, reflect.Float64:
		return rv.Float(), rv.Kind(), true

	default:
		return 0, rv.Kind(), false
	}
}

/**
* numberToInt64: Converts a number to int64
* @param v any
* @return int64, bool
**/
func numberToInt64(v any) (int64, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return 0, false
	}

	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return 0, false
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), true
	default:
		return 0, false
	}
}

/**
* isSignedIntKind
* @param k reflect.Kind
* @return bool
**/
func isSignedIntKind(k reflect.Kind) bool {
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64
}

/**
* isUnsignedIntKind
* @param k reflect.Kind
* @return bool
**/
func isUnsignedIntKind(k reflect.Kind) bool {
	return k == reflect.Uint || k == reflect.Uint8 || k == reflect.Uint16 || k == reflect.Uint32 || k == reflect.Uint64 || k == reflect.Uintptr
}

/**
* matchLikeStar: Matches a string with a pattern
* @param value string, pattern string
* @return bool
**/
func matchLikeStar(value, pattern string) bool {
	// "*" = match todo
	if pattern == "*" {
		return true
	}

	starts := strings.HasPrefix(pattern, "*")
	ends := strings.HasSuffix(pattern, "*")

	core := strings.Trim(pattern, "*")

	// si es "" después de trim (ej: "**") => match todo
	if core == "" {
		return true
	}

	switch {
	// *abc*
	case starts && ends:
		return strings.Contains(value, core)

	// *abc
	case starts && !ends:
		return strings.HasSuffix(value, core)

	// abc*
	case !starts && ends:
		return strings.HasPrefix(value, core)

	// abc (sin comodín)
	default:
		return value == pattern
	}
}

/**
* equalsAny: Compares two values
* @param a any, b any
* @return bool, error
**/
func equalsAny(a, b any) (bool, error) {
	// time.Time
	if ta, ok := a.(time.Time); ok {
		tb, ok := b.(time.Time)
		if !ok {
			return false, nil
		}
		return ta.Equal(tb), nil
	}

	// string
	if sa, ok := a.(string); ok {
		sb, ok := b.(string)
		if !ok {
			return false, nil
		}
		return sa == sb, nil
	}

	// numbers (usa tu helper numberToFloat64 del paso anterior)
	af, _, okA := numberToFloat64(a)
	if okA {
		bf, _, okB := numberToFloat64(b)
		if !okB {
			return false, nil
		}
		return af == bf, nil
	}

	// fallback: solo para tipos comparables
	ra := reflect.ValueOf(a)
	rb := reflect.ValueOf(b)

	if !ra.IsValid() || !rb.IsValid() {
		return false, nil
	}

	// si no son comparables, no se puede hacer ==
	if !ra.Type().Comparable() || !rb.Type().Comparable() {
		return false, nil
	}

	// si son tipos distintos pero comparables, no son iguales
	if ra.Type() != rb.Type() {
		return false, nil
	}

	return ra.Interface() == rb.Interface(), nil
}

/**
* compareAnyOrdered: Compares two values
* @param a any, b any
* @return int, bool
**/
func compareAnyOrdered(a, b any) (int, bool) {
	// time.Time
	if ta, ok := a.(time.Time); ok {
		tb, ok := b.(time.Time)
		if !ok {
			return 0, false
		}
		if ta.Before(tb) {
			return -1, true
		}
		if ta.After(tb) {
			return 1, true
		}
		return 0, true
	}

	// string
	if sa, ok := a.(string); ok {
		sb, ok := b.(string)
		if !ok {
			return 0, false
		}
		if sa < sb {
			return -1, true
		}
		if sa > sb {
			return 1, true
		}
		return 0, true
	}

	// numbers
	af, aKind, okA := numberToFloat64(a)
	if !okA {
		return 0, false
	}

	bf, bKind, okB := numberToFloat64(b)
	if !okB {
		return 0, false
	}

	// Evitar comparar signed vs unsigned si hay negativos (caso peligroso)
	if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
		ai, _ := numberToInt64(a)
		if ai < 0 {
			return 0, false
		}
	}
	if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
		bi, _ := numberToInt64(b)
		if bi < 0 {
			return 0, false
		}
	}

	if af < bf {
		return -1, true
	}
	if af > bf {
		return 1, true
	}
	return 0, true
}

/**
* getBetweenRange: Gets the min and max values from a between range
* @param v any
* @return min any, max any, ok bool
**/
func getBetweenRange(v any) (min any, max any, ok bool) {
	// Caso 1: BetweenValue
	if r, ok := v.(BetweenValue); ok {
		return r.Min, r.Max, true
	}

	// Caso 2: map[string]any {"min":X,"max":Y}
	if m, ok := v.(map[string]any); ok {
		min, okMin := m["min"]
		max, okMax := m["max"]
		return min, max, okMin && okMax
	}

	// Caso 3: slice/array de 2 elementos: []any{min,max}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, nil, false
	}

	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		if rv.Len() != 2 {
			return nil, nil, false
		}
		return rv.Index(0).Interface(), rv.Index(1).Interface(), true
	}

	return nil, nil, false
}

/**
* getField
* @param field string
* @return keys []string, as string
**/
func getField(field string) ([]string, string) {
	if strings.Contains(field, ":") {
		parts := strings.Split(field, ":")
		return strs.Split(parts[0], "->"), parts[1]
	}
	return strs.Split(field, "->"), field
}

/**
* selects
* @param fields []string, object et.Json
* @return et.Json
**/
func selects(fields []string, object et.Json) et.Json {
	result := et.Json{}
	for _, field := range fields {
		keys, as := getField(field)
		val := object.Get(keys...)
		if val != nil {
			result[as] = val
		}
	}

	return result
}

/**
* hidden
* @param fields []string, object et.Json
* @return et.Json
**/
func hidden(fields []string, object et.Json) et.Json {
	result := et.Json{}
	for key, value := range object {
		if slices.Contains(fields, key) {
			continue
		}
		result[key] = value
	}

	return result
}

/**
* MergeToKeyValue
* @params left []et.Json, key string, value any
* @return []et.Json
**/
func MergeToKeyValue(left []et.Json, key string, value any) []et.Json {
	if len(left) == 0 {
		return []et.Json{
			{key: value},
		}
	}

	result := []et.Json{}
	for _, item := range left {
		item[key] = value
		result = append(result, item)
	}

	return result
}

/**
* MergeToMap
* @params left, right []et.Json
* @return []et.Json
**/
func MergeToMap(left, right []et.Json) []et.Json {
	result := []et.Json{}
	for _, itemL := range left {
		for _, itemR := range right {
			maps.Copy(itemR, itemL)
			result = append(result, itemR)
		}
	}

	return result
}

/**
* Prefixer
* @param from *Source
* @return *Source
**/
func Prefixer(item et.Json, as string) et.Json {
	result := et.Json{}
	for k, v := range item {
		k = strings.ToLower(k)
		as := strings.ToLower(as)
		if as != "" {
			result[as+"."+k] = v
		} else {
			result[k] = v
		}
	}

	return result
}

/**
* Joingy
* @params left, right *Source, keys map[string]string, joinType JoinType
* @return *Source
**/
func Joingy(left, right *Source, keys map[string]string, joinType JoinType) *Source {
	result := []et.Json{}
	rightIndex := map[string][]int{}
	matchedRight := map[int]bool{}

	// indexar RIGHT
	for i, r := range right.Data {
		r = Prefixer(r, right.As)
		key := buildKey(r, keys, false)
		rightIndex[key] = append(rightIndex[key], i)
	}

	// recorrer LEFT
	for _, l := range left.Data {
		l = Prefixer(l, left.As)
		key := buildKey(l, keys, true)
		ridxs, ok := rightIndex[key]
		if ok {
			for _, ri := range ridxs {
				r := right.Data[ri]
				r = Prefixer(r, right.As)
				matchedRight[ri] = true

				result = append(result, merge(l, r))
			}
		} else {
			if joinType == LeftJoin || joinType == FullJoin {
				result = append(result, merge(l, nil))
			}
		}
	}

	// RIGHT JOIN o FULL JOIN
	if joinType == RightJoin || joinType == FullJoin {
		for i, r := range right.Data {
			r = Prefixer(r, right.As)
			if !matchedRight[i] {
				result = append(result, merge(nil, r))
			}
		}
	}

	return &Source{
		Data: result,
		As:   left.As,
	}
}

/**
* merge
* @params left, right et.Json, as string
* @return et.Json
**/
func merge(left, right et.Json) et.Json {
	out := et.Json{}

	if left != nil {
		maps.Copy(out, left)
	}

	if right != nil {
		for k, v := range right {
			k = strings.ToLower(k)
			if _, exists := out[k]; exists {
				continue
			}

			out[k] = v
		}
	}

	return out
}

/**
* buildKey
* @params j et.Json, keys map[string]string, fromLeft bool
* @return string
**/
func buildKey(j et.Json, keys map[string]string, fromLeft bool) string {
	key := ""

	for lk, rk := range keys {
		field := rk
		if fromLeft {
			field = lk
		}

		if v, ok := j[field]; ok {
			key += "|" + fmt.Sprintf("%v", v)
		} else {
			key += "|NULL"
		}
	}

	return key
}
