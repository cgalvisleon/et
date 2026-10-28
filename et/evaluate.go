package et

import (
	"cmp"
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

/**
* ApplyToValue: Whether val meets the condition. A FIELD value needs an object to read it from (ApplyToObject), so
* here it is false.
* @param val any
* @return bool
**/
func (s *Condition) ApplyToValue(val any) bool {
	if s.Value.Type == FIELD {
		return false
	}

	val, cv := coerceForComparison(val, s.Value)
	tmp := *s
	tmp.Value = cv

	switch tmp.Operator {
	case EQ:
		return tmp.applyOpEq(val)
	case NEG:
		return tmp.applyOpNeg(val)
	case LESS:
		return tmp.applyOpLess(val)
	case LESS_EQ:
		return tmp.applyOpLessEq(val)
	case MORE:
		return tmp.applyOpMore(val)
	case MORE_EQ:
		return tmp.applyOpMoreEq(val)
	case LIKE:
		return tmp.applyOpLike(val)
	case IN:
		return tmp.applyOpIn(val)
	case NOT_IN:
		return tmp.applyOpNotIn(val)
	case IS:
		return tmp.applyOpIs(val)
	case IS_NOT:
		return !tmp.applyOpIs(val)
	case NULL:
		return tmp.applyOpNull(val)
	case NOT_NULL:
		return tmp.applyOpNotNull(val)
	case BETWEEN:
		return tmp.applyOpBetween(val)
	case NOT_BETWEEN:
		return tmp.applyOpNotBetween(val)
	default:
		return false
	}
}

/**
* ApplyToObject: Whether obj meets the condition. The field may be an aggregate ("sum(items)", applied to the
* attribute) and the value a FIELD (another attribute of obj, also maybe an aggregate).
* @param obj Json
* @return bool
**/
func (s *Condition) ApplyToObject(obj Json) bool {
	var val any
	if fld, ok := ToField(s.Field.String()); ok && fld.Agg != nil {
		val = fld.ValueOf(obj)
	} else {
		val = s.fieldValue(obj)
	}
	return s.resolve(obj).ApplyToValue(val)
}

/**
* evaluate: Combines the conditions with SQL precedence (AND before OR).
* @param conditions []*Condition, apply func(*Condition) bool
* @return bool
**/
func evaluate(conditions []*Condition, apply func(*Condition) bool) bool {
	if len(conditions) == 0 {
		return true
	}

	term := true
	for i, cond := range conditions {
		if i > 0 && cond.Connector == OR {
			if term {
				return true
			}
			term = true
		}
		if term {
			term = apply(cond)
		}
	}

	return term
}

/**
* EvaluateObject: Evaluates the conditions on an object, with SQL precedence.
* @param obj Json, conditions []*Condition
* @return bool
**/
func EvaluateObject(obj Json, conditions []*Condition) bool {
	return evaluate(conditions, func(cond *Condition) bool { return cond.ApplyToObject(obj) })
}

/**
* EvaluateValue: Evaluates the conditions on a value, with SQL precedence.
* @param value any, conditions []*Condition
* @return bool
**/
func EvaluateValue(value any, conditions []*Condition) bool {
	return evaluate(conditions, func(cond *Condition) bool { return cond.ApplyToValue(value) })
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
			if f, ok := floatOfAny(item); ok {
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
		if f, ok := floatOfAny(val); ok {
			return int64(f)
		}
		return nil
	case FLOAT_VALUE:
		if f, ok := floatOfAny(val); ok {
			return f
		}
		return nil
	}
	return nil
}

/**
* fieldValue
* @param data Json
* @return any, error
**/
func (s *Condition) fieldValue(data Json) (result any) {
	result = data
	fields := s.Field.Fields()
	for _, field := range fields {
		switch v := result.(type) {
		case Json:
			val, ok := v[field]
			if !ok {
				result = nil
				return
			}

			result = val
		case map[string]interface{}:
			val, ok := v[field]
			if !ok {
				result = nil
				return
			}

			result = val
		default:
			result = nil
			return
		}
	}

	return
}

/**
* applyOpEq
* @param val any
* @return bool
**/
func (s *Condition) applyOpEq(val any) bool {
	if val == nil {
		return false
	}

	switch bv := s.Value.Value.(type) {
	case []Json:
		if len(bv) == 0 {
			return false
		}
		value, ok := firstMapValueSorted(bv[0])
		if !ok {
			return false
		}
		result, err := equalsAny(val, value)
		if err != nil {
			return false
		}
		return result
	default:
		if first, ok := firstOfSlice(bv); ok {
			tmp := *s
			tmp.Value = NewValue(first)
			return tmp.applyOpEq(val)
		}

		ok, err := equalsAny(val, bv)
		if err != nil {
			return false
		}
		return ok
	}
}

/**
* applyOpNeg
* @param val any
* @return bool
**/
func (s *Condition) applyOpNeg(val any) bool {
	return !s.applyOpEq(val)
}

/**
* applyOpOrdered: Compares val with the condition value (times, texts or numbers, integers exactly) and tests the
* result; a list value takes its first element.
* @param val any, test func(int) bool
* @return bool
**/
func (s *Condition) applyOpOrdered(val any, test func(int) bool) bool {
	if val == nil {
		return false
	}

	switch bv := s.Value.Value.(type) {
	case []Json:
		if len(bv) == 0 {
			return false
		}
		value, ok := firstMapValueSorted(bv[0])
		if !ok {
			return false
		}
		tmp := *s
		tmp.Value = NewValue(value)
		return tmp.applyOpOrdered(val, test)
	default:
		if first, ok := firstOfSlice(bv); ok {
			tmp := *s
			tmp.Value = NewValue(first)
			return tmp.applyOpOrdered(val, test)
		}
		c, ok := compareAnyOrdered(val, bv)
		return ok && test(c)
	}
}

/**
* applyOpLess
* @param val any
* @return bool
**/
func (s *Condition) applyOpLess(val any) bool {
	return s.applyOpOrdered(val, func(c int) bool { return c < 0 })
}

/**
* applyOpLessEq
* @param val any
* @return bool
**/
func (s *Condition) applyOpLessEq(val any) bool {
	return s.applyOpOrdered(val, func(c int) bool { return c <= 0 })
}

/**
* applyOpMore
* @param val any
* @return bool
**/
func (s *Condition) applyOpMore(val any) bool {
	return s.applyOpOrdered(val, func(c int) bool { return c > 0 })
}

/**
* applyOpMoreEq
* @param val any
* @return bool
**/
func (s *Condition) applyOpMoreEq(val any) bool {
	return s.applyOpOrdered(val, func(c int) bool { return c >= 0 })
}

/**
* applyOpLike
* @param val any
* @return bool
**/
func (s *Condition) applyOpLike(val any) bool {
	if val == nil {
		return false
	}

	invalidType := func() bool {
		return false
	}

	switch bv := s.Value.Value.(type) {
	case string:
		av, ok := val.(string)
		if !ok {
			return invalidType()
		}
		return MatchLike(av, bv)
	case Json:
		value, ok := firstMapValueSorted(bv)
		if !ok {
			return invalidType()
		}
		tmp := *s
		tmp.Value = NewValue(value)
		return tmp.applyOpLike(val)
	case map[string]interface{}:
		value, ok := firstMapValueSorted(bv)
		if !ok {
			return invalidType()
		}
		tmp := *s
		tmp.Value = NewValue(value)
		return tmp.applyOpLike(val)
	case []Json:
		if len(bv) == 0 {
			return invalidType()
		}
		value, ok := firstMapValueSorted(bv[0])
		if !ok {
			return invalidType()
		}
		tmp := *s
		tmp.Value = NewValue(value)
		return tmp.applyOpLike(val)
	case []interface{}:
		for _, value := range bv {
			tmp := *s
			tmp.Value = NewValue(value)
			return tmp.applyOpLike(val)
		}
		return invalidType()
	default:
		if first, ok := firstOfSlice(bv); ok {
			tmp := *s
			tmp.Value = NewValue(first)
			return tmp.applyOpLike(val)
		}
		return invalidType()
	}
}

/**
* applyOpIn
* @param val any
* @return bool
**/
func (s *Condition) applyOpIn(val any) bool {
	if val == nil {
		return false
	}

	invalidType := func() bool {
		return false
	}

	list := reflect.ValueOf(s.Value.Value)
	if !list.IsValid() {
		return invalidType()
	}

	if list.Kind() != reflect.Slice && list.Kind() != reflect.Array {
		return invalidType()
	}

	for i := 0; i < list.Len(); i++ {
		item := list.Index(i).Interface()

		ok, err := equalsAny(val, item)
		if err != nil {
			return false
		}
		if ok {
			return true
		}
	}

	return false
}

/**
* applyOpNotIn
* @param val any
* @return bool
**/
func (s *Condition) applyOpNotIn(val any) bool {
	ok := s.applyOpIn(val)
	return !ok
}

/**
* applyOpIs
* @param val any
* @return bool
**/
func (s *Condition) applyOpIs(val any) bool {
	if val == nil && s.Value.Value == nil {
		return true
	}

	if val == nil || s.Value.Value == nil {
		return false
	}

	ok, err := equalsAny(val, s.Value.Value)
	if err != nil {
		return false
	}
	return ok
}

/**
* applyOpNull
* @param val any
* @return bool
**/
func (s *Condition) applyOpNull(val any) bool {
	return val == nil
}

/**
* applyOpNotNull
* @param val any
* @return bool
**/
func (s *Condition) applyOpNotNull(val any) bool {
	ok := s.applyOpNull(val)
	return !ok
}

/**
* applyOpBetween
* @param val any
* @return bool
**/
func (s *Condition) applyOpBetween(val any) bool {
	in, ok := s.between(val)
	return ok && in
}

/**
* applyOpNotBetween: As SQL, nil or an unreadable range is false
* @param val any
* @return bool
**/
func (s *Condition) applyOpNotBetween(val any) bool {
	in, ok := s.between(val)
	return ok && !in
}

/**
* between: Whether val is inside the range; ok is false with nil or a range that does not compare
* @param val any
* @return in bool, ok bool
**/
func (s *Condition) between(val any) (in bool, ok bool) {
	if val == nil {
		return false, false
	}

	bv, ok := ToBetween(s.Value.Value)
	if !ok || bv.Min == nil || bv.Max == nil {
		return false, false
	}

	c1, ok := compareAnyOrdered(val, bv.Min)
	if !ok {
		return false, false
	}

	c2, ok := compareAnyOrdered(val, bv.Max)
	if !ok {
		return false, false
	}

	return c1 >= 0 && c2 <= 0, true
}

/**
* ToBetween: Reads the range: a BetweenValue, {min, max} or [min, max]
* @param v any
* @return BetweenValue, bool
**/
func ToBetween(v any) (BetweenValue, bool) {
	min, max, ok := getBetweenRange(v)
	return BetweenValue{Min: min, Max: max}, ok
}

/**
* getBetweenRange: Gets the min and max values from a between range
* @param v any
* @return min any, max any, ok bool
**/
func getBetweenRange(v any) (min any, max any, ok bool) {
	if r, ok := v.(BetweenValue); ok {
		return r.Min, r.Max, true
	}

	// {min, max} or {Min, Max}
	if m, ok := asJson(v); ok {
		min, okMin := m["min"]
		if !okMin {
			min, okMin = m["Min"]
		}
		max, okMax := m["max"]
		if !okMax {
			max, okMax = m["Max"]
		}
		return min, max, okMin && okMax
	}

	// [min, max]
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
* coerceForComparison: Normalizes val and the condition value so that a
* datetime can be compared against its string representation, regardless of
* which side carries the time.Time and which carries the string.
* @param val any, cv Value
* @return any, Value
**/
func coerceForComparison(val any, cv Value) (any, Value) {
	switch a := val.(type) {
	case time.Time:
		if b, ok := cv.Value.(string); ok {
			if t, ok := parseDate(b); ok {
				return val, Value{Type: DATETIME, Value: t}
			}
		}
	case string:
		if _, ok := cv.Value.(time.Time); ok {
			if t, ok := parseDate(a); ok {
				return t, cv
			}
		}
	}

	return val, cv
}

/**
* ApplyToIndex
* @param keys []string
* @return []string
**/
func (s *Condition) ApplyToIndex(keys []string) []string {
	result := make([]string, 0)
	fields := s.Field.Fields()
	if len(fields) == 0 {
		return result
	}

	for _, key := range keys {
		ok := s.ApplyToValue(key)
		if ok {
			result = append(result, key)
		}
	}

	return result
}

/**
* numberKind: How a number is held for an exact comparison.
**/
type numberKind int

const (
	numSigned numberKind = iota
	numUnsigned
	numFloat
)

/**
* number: A number as an int64, uint64 or float64, without losing precision.
**/
type number struct {
	kind numberKind
	i    int64
	u    uint64
	f    float64
}

/**
* numberOf: Reads v as a number keeping integers exact (json.Number and pointers included).
* @param v any
* @return number, bool
**/
func numberOf(v any) (number, bool) {
	if n, ok := v.(json.Number); ok {
		if i, err := n.Int64(); err == nil {
			return number{kind: numSigned, i: i}, true
		}
		if u, err := strconv.ParseUint(string(n), 10, 64); err == nil {
			return number{kind: numUnsigned, u: u}, true
		}
		f, err := n.Float64()
		return number{kind: numFloat, f: f}, err == nil
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return number{}, false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return number{}, false
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return number{kind: numSigned, i: rv.Int()}, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return number{kind: numUnsigned, u: rv.Uint()}, true
	case reflect.Float32, reflect.Float64:
		return number{kind: numFloat, f: rv.Float()}, true
	default:
		return number{}, false
	}
}

/**
* exact: The number as an integer when it is one that fits in int64 or uint64 (a float with no decimals too).
* @return number, bool
**/
func (n number) exact() (number, bool) {
	if n.kind != numFloat {
		return n, true
	}
	if n.f != math.Trunc(n.f) {
		return n, false
	}
	if n.f >= -(1<<63) && n.f < 1<<63 {
		return number{kind: numSigned, i: int64(n.f)}, true
	}
	if n.f >= 0 && n.f < 1<<64 {
		return number{kind: numUnsigned, u: uint64(n.f)}, true
	}
	return n, false
}

/**
* float: The number as float64.
* @return float64
**/
func (n number) float() float64 {
	switch n.kind {
	case numSigned:
		return float64(n.i)
	case numUnsigned:
		return float64(n.u)
	default:
		return n.f
	}
}

/**
* compareNumbers: Compares two numbers, integers exactly; NaN does not compare.
* @param a any, b any
* @return int, bool
**/
func compareNumbers(a, b any) (int, bool) {
	na, ok := numberOf(a)
	if !ok {
		return 0, false
	}
	nb, ok := numberOf(b)
	if !ok {
		return 0, false
	}
	if math.IsNaN(na.float()) || math.IsNaN(nb.float()) {
		return 0, false
	}

	ea, okA := na.exact()
	eb, okB := nb.exact()
	if !okA || !okB {
		return cmp.Compare(na.float(), nb.float()), true
	}

	switch {
	case ea.kind == numSigned && eb.kind == numSigned:
		return cmp.Compare(ea.i, eb.i), true
	case ea.kind == numUnsigned && eb.kind == numUnsigned:
		return cmp.Compare(ea.u, eb.u), true
	case ea.kind == numSigned:
		if ea.i < 0 {
			return -1, true
		}
		return cmp.Compare(uint64(ea.i), eb.u), true
	default:
		if eb.i < 0 {
			return 1, true
		}
		return cmp.Compare(ea.u, uint64(eb.i)), true
	}
}

/**
* MatchLike: SQL LIKE, case insensitive (%, _, \\ escapes; * as %).
* @param value string, pattern string
* @return bool
**/
func MatchLike(value, pattern string) bool {
	v := []rune(strings.ToLower(value))
	type token struct {
		r   rune
		any bool // % or *
		one bool // _
	}
	tokens := []token{}
	p := []rune(strings.ToLower(pattern))
	for i := 0; i < len(p); i++ {
		switch r := p[i]; {
		case r == '\\' && i+1 < len(p):
			i++
			tokens = append(tokens, token{r: p[i]})
		case r == '%' || r == '*':
			if n := len(tokens); n == 0 || !tokens[n-1].any {
				tokens = append(tokens, token{any: true})
			}
		case r == '_':
			tokens = append(tokens, token{one: true})
		default:
			tokens = append(tokens, token{r: r})
		}
	}

	// Greedy match with backtracking to the last %
	vi, ti, star, mark := 0, 0, -1, 0
	for vi < len(v) {
		switch {
		case ti < len(tokens) && !tokens[ti].any && (tokens[ti].one || tokens[ti].r == v[vi]):
			vi++
			ti++
		case ti < len(tokens) && tokens[ti].any:
			star, mark = ti, vi
			ti++
		case star >= 0:
			mark++
			vi, ti = mark, star+1
		default:
			return false
		}
	}
	for ti < len(tokens) && tokens[ti].any {
		ti++
	}
	return ti == len(tokens)
}

/**
* equalsAny: Compares two values
* @param a any, b any
* @return bool, error
**/
func equalsAny(a, b any) (bool, error) {
	if ta, ok := a.(time.Time); ok {
		tb, ok := b.(time.Time)
		if !ok {
			return false, nil
		}
		return ta.Equal(tb), nil
	}

	if sa, ok := a.(string); ok {
		sb, ok := b.(string)
		if !ok {
			return false, nil
		}
		return sa == sb, nil
	}

	if _, ok := numberOf(a); ok {
		c, ok := compareNumbers(a, b)
		return ok && c == 0, nil
	}

	// fallback: comparable values of the same type
	ra := reflect.ValueOf(a)
	rb := reflect.ValueOf(b)

	if !ra.IsValid() || !rb.IsValid() {
		return false, nil
	}

	if !ra.Type().Comparable() || !rb.Type().Comparable() {
		return false, nil
	}

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

	return compareNumbers(a, b)
}

/**
* firstOfSlice: Returns the first element of a slice/array value.
* @param v any
* @return any, bool
**/
func firstOfSlice(v any) (any, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, false
	}

	if (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) || rv.Len() == 0 {
		return nil, false
	}

	return rv.Index(0).Interface(), true
}

/**
* firstMapValueSorted: The value of the alphabetically first key of m, so unwrapping a map is deterministic
* (range order is random); false when m is empty.
* @param m map[string]interface{}
* @return any, bool
**/
func firstMapValueSorted(m map[string]interface{}) (any, bool) {
	if len(m) == 0 {
		return nil, false
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return m[keys[0]], true
}
