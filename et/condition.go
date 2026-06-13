package et

import (
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/cgalvisleon/et/msg"
)

var (
	ErrorFieldNotFound = errors.New(msg.MSG_FIELD_NOT_FOUND)
	ErrorDataNotFound  = errors.New(msg.MSG_DATA_NOT_FOUND)
)

type Operator string

const (
	EQ          Operator = "eq"
	NEG         Operator = "neg"
	LESS        Operator = "less"
	LESS_EQ     Operator = "less_eq"
	MORE        Operator = "more"
	MORE_EQ     Operator = "more_eq"
	LIKE        Operator = "like"
	IN          Operator = "in"
	NOT_IN      Operator = "not_in"
	IS          Operator = "is"
	IS_NOT      Operator = "is_not"
	NULL        Operator = "null"
	NOT_NULL    Operator = "not_null"
	BETWEEN     Operator = "between"
	NOT_BETWEEN Operator = "not_between"
)

func (s Operator) Str() string {
	return string(s)
}

func ToOperator(s string) Operator {
	values := map[string]Operator{
		"eq":          EQ,
		"neg":         NEG,
		"less":        LESS,
		"less_eq":     LESS_EQ,
		"more":        MORE,
		"more_eq":     MORE_EQ,
		"like":        LIKE,
		"in":          IN,
		"not_in":      NOT_IN,
		"is":          IS,
		"is_not":      IS_NOT,
		"null":        NULL,
		"not_null":    NOT_NULL,
		"between":     BETWEEN,
		"not_between": NOT_BETWEEN,
	}

	result, ok := values[s]
	if !ok {
		return EQ
	}

	return result
}

type Connector string

const (
	NaC Connector = ""
	And Connector = "and"
	Or  Connector = "or"
)

func (s Connector) Str() string {
	return string(s)
}

type BetweenValue struct {
	Min any `json:"Min"`
	Max any `json:"Max"`
}

const (
	ValueString   = "string"
	ValueInt      = "int"
	ValueFloat    = "float"
	ValueBool     = "bool"
	ValueDatetime = "datetime"
	ValueArray    = "array"
	ValueJson     = "json"
	ValueBetween  = "between"
	ValueNull     = "null"
	ValueAny      = "any"
)

type Value struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}

/**
* Raw: Returns the underlying raw value.
* @return any
**/
func (v Value) Raw() any {
	return v.Value
}

/**
* NewValue: Wraps a raw value into a Value, inferring its logical Type.
* @param v any
* @return Value
**/
func NewValue(v any) Value {
	return Value{Type: valueType(v), Value: v}
}

/**
* valueType: Infers the logical type name for a raw value.
* @param v any
* @return string
**/
func valueType(v any) string {
	switch v.(type) {
	case nil:
		return ValueNull
	case string:
		return ValueString
	case bool:
		return ValueBool
	case time.Time:
		return ValueDatetime
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return ValueInt
	case float32, float64:
		return ValueFloat
	case BetweenValue:
		return ValueBetween
	case []Json, []interface{}:
		return ValueArray
	case Json, map[string]interface{}:
		return ValueJson
	default:
		return ValueAny
	}
}

type Condition struct {
	Field     string    `json:"field"`
	Operator  Operator  `json:"operator"`
	Value     Value     `json:"value"`
	Connector Connector `json:"connector"`
}

/**
* ToJson
* @return Json
**/
func (s *Condition) ToJson() Json {
	if s.Connector == NaC {
		return Json{
			s.Field: Json{
				s.Operator.Str(): s.Value.Value,
			},
		}
	}

	return Json{
		s.Connector.Str(): Json{
			s.Field: Json{
				s.Operator.Str(): s.Value.Value,
			},
		},
	}
}

/**
* fieldValue
* @param data Json
* @return any, error
**/
func (s *Condition) fieldValue(data Json) (result any) {
	result = data.Clone()
	fields := strings.Split(s.Field, "->")
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
		for _, item := range bv {
			for _, value := range item {
				ok, err := equalsAny(val, value)
				if err != nil {
					return false
				}
				return ok
			}
		}
		return false
	default:
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
* applyOpLess
* @param val any
* @return bool
**/
func (s *Condition) applyOpLess(val any) bool {
	if val == nil {
		return false
	}

	invalidType := func() bool {
		return false
	}

	switch bv := s.Value.Value.(type) {
	case time.Time:
		if av, ok := val.(time.Time); ok {
			return av.Before(bv)
		}
		return invalidType()
	case string:
		if av, ok := val.(string); ok {
			return av < bv
		}
		return invalidType()
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		aNum, aKind, ok := numberToFloat64(val)
		if !ok {
			return invalidType()
		}

		bNum, bKind, ok := numberToFloat64(s.Value.Value)
		if !ok {
			return invalidType()
		}

		if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
			ai, _ := numberToInt64(val)
			if ai < 0 {
				return invalidType()
			}
		}
		if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
			bi, _ := numberToInt64(s.Value.Value)
			if bi < 0 {
				return invalidType()
			}
		}

		return aNum < bNum
	case []Json:
		for _, item := range bv {
			for _, value := range item {
				tmp := *s
				tmp.Value = NewValue(value)
				return tmp.applyOpLess(val)
			}
		}
		return invalidType()
	case []interface{}:
		for _, value := range bv {
			tmp := *s
			tmp.Value = NewValue(value)
			return tmp.applyOpLess(val)
		}
		return invalidType()
	default:
		return invalidType()
	}
}

/**
* applyOpLessEq
* @param val any
* @return bool
**/
func (s *Condition) applyOpLessEq(val any) bool {
	if val == nil {
		return false
	}

	invalidType := func() bool {
		return false
	}

	switch bv := s.Value.Value.(type) {
	case time.Time:
		if av, ok := val.(time.Time); ok {
			return av.Before(bv) || av.Equal(bv)
		}
		return invalidType()
	case string:
		if av, ok := val.(string); ok {
			return av <= bv
		}
		return invalidType()
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		aNum, aKind, ok := numberToFloat64(val)
		if !ok {
			return invalidType()
		}

		bNum, bKind, ok := numberToFloat64(s.Value.Value)
		if !ok {
			return invalidType()
		}

		if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
			ai, _ := numberToInt64(val)
			if ai < 0 {
				return invalidType()
			}
		}
		if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
			bi, _ := numberToInt64(s.Value.Value)
			if bi < 0 {
				return invalidType()
			}
		}

		return aNum <= bNum
	case []Json:
		for _, item := range bv {
			for _, value := range item {
				tmp := *s
				tmp.Value = NewValue(value)
				return tmp.applyOpLessEq(val)
			}
		}
		return invalidType()
	case []interface{}:
		for _, value := range bv {
			tmp := *s
			tmp.Value = NewValue(value)
			return tmp.applyOpLessEq(val)
		}
		return invalidType()
	default:
		return invalidType()
	}
}

/**
* applyOpMore
* @param val any
* @return bool
**/
func (s *Condition) applyOpMore(val any) bool {
	if val == nil {
		return false
	}

	invalidType := func() bool {
		return false
	}

	switch bv := s.Value.Value.(type) {
	case time.Time:
		if av, ok := val.(time.Time); ok {
			return av.After(bv)
		}
		return invalidType()
	case string:
		if av, ok := val.(string); ok {
			return av > bv
		}
		return invalidType()
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		aNum, aKind, ok := numberToFloat64(val)
		if !ok {
			return invalidType()
		}

		bNum, bKind, ok := numberToFloat64(s.Value.Value)
		if !ok {
			return invalidType()
		}

		if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
			ai, _ := numberToInt64(val)
			if ai < 0 {
				return invalidType()
			}
		}
		if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
			bi, _ := numberToInt64(s.Value.Value)
			if bi < 0 {
				return invalidType()
			}
		}

		return aNum > bNum
	case []Json:
		for _, item := range bv {
			for _, value := range item {
				tmp := *s
				tmp.Value = NewValue(value)
				return tmp.applyOpMore(val)
			}
		}
		return invalidType()
	case []interface{}:
		for _, value := range bv {
			tmp := *s
			tmp.Value = NewValue(value)
			return tmp.applyOpMore(val)
		}
		return invalidType()
	default:
		return invalidType()
	}
}

/**
* applyOpMoreEq
* @param val any
* @return bool
**/
func (s *Condition) applyOpMoreEq(val any) bool {
	if val == nil {
		return false
	}

	invalidType := func() bool {
		return false
	}

	switch bv := s.Value.Value.(type) {
	case time.Time:
		if av, ok := val.(time.Time); ok {
			return av.After(bv) || av.Equal(bv)
		}
		return invalidType()
	case string:
		if av, ok := val.(string); ok {
			return av >= bv
		}
		return invalidType()
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		aNum, aKind, ok := numberToFloat64(val)
		if !ok {
			return invalidType()
		}

		bNum, bKind, ok := numberToFloat64(s.Value.Value)
		if !ok {
			return invalidType()
		}

		if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
			ai, _ := numberToInt64(val)
			if ai < 0 {
				return invalidType()
			}
		}
		if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
			bi, _ := numberToInt64(s.Value.Value)
			if bi < 0 {
				return invalidType()
			}
		}

		return aNum >= bNum
	case []Json:
		for _, item := range bv {
			for _, value := range item {
				tmp := *s
				tmp.Value = NewValue(value)
				return tmp.applyOpMoreEq(val)
			}
		}
		return invalidType()
	case []interface{}:
		for _, value := range bv {
			tmp := *s
			tmp.Value = NewValue(value)
			return tmp.applyOpMoreEq(val)
		}
		return invalidType()
	default:
		return invalidType()
	}
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
		return matchLikeStar(av, bv)
	case Json:
		for _, value := range bv {
			tmp := *s
			tmp.Value = NewValue(value)
			return tmp.applyOpLike(val)
		}
		return invalidType()
	case map[string]interface{}:
		for _, value := range bv {
			tmp := *s
			tmp.Value = NewValue(value)
			return tmp.applyOpLike(val)
		}
		return invalidType()
	case []Json:
		for _, item := range bv {
			for _, value := range item {
				tmp := *s
				tmp.Value = NewValue(value)
				return tmp.applyOpLike(val)
			}
		}
		return invalidType()
	case []interface{}:
		for _, value := range bv {
			tmp := *s
			tmp.Value = NewValue(value)
			return tmp.applyOpLike(val)
		}
		return invalidType()
	default:
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
	if val == nil {
		return false
	}

	min, max, ok := getBetweenRange(s.Value.Value)
	if !ok {
		return false
	}

	if min == nil || max == nil {
		return false
	}

	c1, ok := compareAnyOrdered(val, min)
	if !ok {
		return false
	}

	c2, ok := compareAnyOrdered(val, max)
	if !ok {
		return false
	}

	return c1 >= 0 && c2 <= 0
}

/**
* applyOpNotBetween
* @param val any
* @return bool
**/
func (s *Condition) applyOpNotBetween(val any) bool {
	ok := s.applyOpBetween(val)
	return !ok
}

/**
* dateLayouts: Layouts tried when coercing a string into a time.Time for comparison.
**/
var dateLayouts = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02 15:04:05",
	"2006-01-02",
}

/**
* parseDate: Tries to parse a string into a time.Time using dateLayouts.
* @param s string
* @return time.Time, bool
**/
func parseDate(s string) (time.Time, bool) {
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}

	return time.Time{}, false
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
				return val, Value{Type: ValueDatetime, Value: t}
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
* ApplyToValue
* @param val any
* @return bool
**/
func (s *Condition) ApplyToValue(val any) bool {
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
* ApplyToObject
* @param obj Json
* @return bool
**/
func (s *Condition) ApplyToObject(obj Json) bool {
	val := s.fieldValue(obj)
	return s.ApplyToValue(val)
}

/**
* ApplyToIndex
* @param keys []string
* @return []string
**/
func (s *Condition) ApplyToIndex(keys []string) []string {
	result := make([]string, 0)
	if s.Field == "" {
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
* ToCondition
* @param json Json
* @return []*Condition
**/
func ToCondition(json Json) []*Condition {
	result := []*Condition{}

	getWhere := func(json Json) *Condition {
		for fld := range json {
			cond := json.Json(fld)
			for cnd := range cond {
				val := cond[cnd]
				return condition(fld, val, ToOperator(cnd))
			}
		}
		return nil
	}

	and := func(jsons Json) *Condition {
		result := getWhere(jsons)
		if result != nil {
			result.Connector = And
		}

		return result
	}

	or := func(jsons Json) *Condition {
		result := getWhere(jsons)
		if result != nil {
			result.Connector = Or
		}

		return result
	}

	for k := range json {
		if strings.ToLower(k) == "and" {
			def := json.Json(k)
			result = append(result, and(def))
		} else if strings.ToLower(k) == "or" {
			def := json.Json(k)
			result = append(result, or(def))
		} else if strings.ToLower(k) == "where" {
			def := json.Json(k)
			result = append(result, getWhere(def))
		} else if strings.ToLower(k) == "on" {
			def := json.Json(k)
			result = append(result, getWhere(def))
		}
	}

	return result
}

/**
* condition
* @param field string, value interface{}, op string
* @return *Condition
**/
func condition(field string, value interface{}, op Operator) *Condition {
	return &Condition{
		Field:     field,
		Operator:  op,
		Value:     NewValue(value),
		Connector: NaC,
	}
}

/**
* Eq
* @param field string, value interface{}
* @return Condition
**/
func Eq(field string, value interface{}) *Condition {
	return condition(field, value, EQ)
}

/**
* Neg
* @param field string, value interface{}
* @return Condition
**/
func Neg(field string, value interface{}) *Condition {
	return condition(field, value, NEG)
}

/**
* Less
* @param field string, value interface{}
* @return Condition
**/
func Less(field string, value interface{}) *Condition {
	return condition(field, value, LESS)
}

/**
* LessEq
* @param field string, value interface{}
* @return Condition
**/
func LessEq(field string, value interface{}) *Condition {
	return condition(field, value, LESS_EQ)
}

/**
* More
* @param field string, value interface{}
* @return Condition
**/
func More(field string, value interface{}) *Condition {
	return condition(field, value, MORE)
}

/**
* MoreEq
* @param field string, value interface{}
* @return Condition
**/
func MoreEq(field string, value interface{}) *Condition {
	return condition(field, value, MORE_EQ)
}

/**
* Like
* @param field string, value interface{}
* @return Condition
**/
func Like(field string, value interface{}) *Condition {
	return condition(field, value, LIKE)
}

/**
* In
* @param field string, value []interface{}
* @return Condition
**/
func In(field string, value []interface{}) *Condition {
	return condition(field, value, IN)
}

/**
* NotIn
* @param field string, value []interface{}
* @return Condition
**/
func NotIn(field string, value []interface{}) *Condition {
	return condition(field, value, NOT_IN)
}

/**
* Is
* @param field string, value interface{}
* @return Condition
**/
func Is(field string, value interface{}) *Condition {
	return condition(field, value, IS)
}

/**
* IsNot
* @param field string, value interface{}
* @return Condition
**/
func IsNot(field string, value interface{}) *Condition {
	return condition(field, value, IS_NOT)
}

/**
* Null
* @param field string
* @return Condition
**/
func Null(field string) *Condition {
	return condition(field, nil, NULL)
}

/**
* NotNull
* @param field string
* @return Condition
**/
func NotNull(field string) *Condition {
	return condition(field, nil, NOT_NULL)
}

/**
* Between
* @param field string, min any, max any
* @return Condition
**/
func Between(field string, min, max any) *Condition {
	return condition(field, BetweenValue{Min: min, Max: max}, BETWEEN)
}

/**
* NotBetween
* @param field string, min any, max any
* @return Condition
**/
func NotBetween(field string, min, max any) *Condition {
	return condition(field, BetweenValue{Min: min, Max: max}, NOT_BETWEEN)
}

/**
* Evaluate
* @param item Json, conditions []*Condition
* @return bool
**/
func Evaluate(item Json, conditions []*Condition) bool {
	if len(conditions) == 0 {
		return true
	}

	var result bool
	for i, con := range conditions {
		ok := con.ApplyToObject(item)
		if i == 0 {
			result = ok
			continue
		}

		if con.Connector == And {
			result = result && ok
		} else if con.Connector == Or {
			result = result || ok
		}

		if !result {
			break
		}
	}

	return result
}
