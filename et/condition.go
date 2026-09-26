package et

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrorFieldNotFound = errors.New(MSG_FIELD_NOT_FOUND)
	ErrorDataNotFound  = errors.New(MSG_DATA_NOT_FOUND)
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
	s = strings.ToLower(s)
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

/**
* Time: Converts a value to a time.Time
* @param val any
* @return time.Time
**/
func Time(val any) *time.Time {
	switch v := val.(type) {
	case time.Time:
		return &v
	case *time.Time:
		return v
	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return nil
		}
		return &t
	default:
		return nil
	}
}

type TypeData string

const (
	ANY            TypeData = "any"
	BYTE           TypeData = "byte"
	KEY            TypeData = "key"
	TEXT           TypeData = "text"
	MEMO           TypeData = "memo"
	INT            TypeData = "int"
	FLOAT          TypeData = "float"
	BOOL           TypeData = "bool"
	DATETIME       TypeData = "datetime"
	JSON           TypeData = "json"
	ARRAY          TypeData = "array"
	ARRAY_JSON     TypeData = "array_json"
	ARRAY_STRING   TypeData = "array_string"
	ARRAY_INT      TypeData = "array_int"
	ARRAY_FLOAT    TypeData = "array_float"
	ARRAY_BOOL     TypeData = "array_bool"
	ARRAY_DATETIME TypeData = "array_datetime"
	VAL_BETWEEN    TypeData = "between"
	VAL_NULL       TypeData = "null"
	AGGREGATE      TypeData = "aggregate"
)

func (s TypeData) Str() string {
	return string(s)
}

var TypeValues = map[string]TypeData{
	ANY.Str():            ANY,
	BYTE.Str():           BYTE,
	KEY.Str():            KEY,
	TEXT.Str():           TEXT,
	MEMO.Str():           MEMO,
	INT.Str():            INT,
	FLOAT.Str():          FLOAT,
	BOOL.Str():           BOOL,
	DATETIME.Str():       DATETIME,
	JSON.Str():           JSON,
	ARRAY.Str():          ARRAY,
	ARRAY_JSON.Str():     ARRAY_JSON,
	ARRAY_STRING.Str():   ARRAY_STRING,
	ARRAY_INT.Str():      ARRAY_INT,
	ARRAY_FLOAT.Str():    ARRAY_FLOAT,
	ARRAY_BOOL.Str():     ARRAY_BOOL,
	ARRAY_DATETIME.Str(): ARRAY_DATETIME,
	VAL_BETWEEN.Str():    VAL_BETWEEN,
	VAL_NULL.Str():       VAL_NULL,
	AGGREGATE.Str():      AGGREGATE,
}

func IsTypeValue(v string) bool {
	v = strings.ToLower(v)
	_, exists := TypeValues[v]
	return exists
}

type Value struct {
	Type  TypeData `json:"type"`
	Value any      `json:"value"`
}

/**
* Raw: Returns the underlying raw value.
* @return any
**/
func (v Value) Raw() any {
	return v.Value
}

/**
* String: Returns the string representation of the value.
* @return string
**/
func (v Value) String() string {
	return fmt.Sprintf("%v", v.Value)
}

/**
* Is: Checks if the value is of the given type.
* @param tpData string
* @return bool
**/
func (v Value) Is(tpData string) bool {
	if !IsTypeValue(tpData) {
		return false
	}
	tpData = strings.ToLower(tpData)
	return v.Type == TypeData(tpData)
}

/**
* Fields: Returns the fields of the value.
* @return []string
**/
func (v Value) Fields() []string {
	if v.Type != TEXT {
		return []string{}
	}
	return strings.Split(v.String(), "->")
}

/**
* NewValue: Wraps a raw value into a Value, inferring its logical Type.
* @param v any
* @return Value
**/
func NewValue(v any) Value {
	switch t := v.(type) {
	case Value:
		return t
	default:
		return Value{Type: valueType(v), Value: v}
	}
}

/**
* valueType: Infers the logical type name for a raw value.
* @param v any
* @return string
**/
func valueType(v any) TypeData {
	switch v.(type) {
	case nil:
		return VAL_NULL
	case string:
		if len(v.(string)) > 255 {
			return MEMO
		}
		return TEXT
	case []byte:
		return BYTE
	case bool:
		return BOOL
	case time.Time:
		return DATETIME
	case *time.Time:
		return DATETIME
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return INT
	case float32, float64:
		return FLOAT
	case Json, map[string]interface{}:
		return JSON
	case BetweenValue:
		return VAL_BETWEEN
	case Aggregate, *Aggregate:
		return AGGREGATE
	case []interface{}:
		return ARRAY
	case []Json:
		return ARRAY_JSON
	case []string:
		return ARRAY_STRING
	case []int, []int8, []int16, []int32, []int64, []uint, []uint16, []uint32, []uint64:
		return ARRAY_INT
	case []float32, []float64:
		return ARRAY_FLOAT
	case []bool:
		return ARRAY_BOOL
	case []time.Time:
		return ARRAY_DATETIME
	default:
		return TypeData(fmt.Sprintf("%T", v))
	}
}

type BetweenValue struct {
	Min any `json:"Min"`
	Max any `json:"Max"`
}

type Function int

const (
	COUNT Function = iota
	SUM
	AVG
	MIN
	MAX
	EXP
	EXTRACT_YEAR
	EXTRACT_MONTH
	EXTRACT_DAY
	EXTRACT_HOUR
	EXTRACT_MINUTE
	EXTRACT_SECOND
	INT_VALUE
	FLOAT_VALUE
)

func ToFunction(s string) Function {
	switch s {
	case "count":
		return COUNT
	case "sum":
		return SUM
	case "avg":
		return AVG
	case "min":
		return MIN
	case "max":
		return MAX
	case "exp":
		return EXP
	case "extract_year":
		return EXTRACT_YEAR
	case "extract_month":
		return EXTRACT_MONTH
	case "extract_day":
		return EXTRACT_DAY
	case "extract_hour":
		return EXTRACT_HOUR
	case "extract_minute":
		return EXTRACT_MINUTE
	case "extract_second":
		return EXTRACT_SECOND
	case "int_value":
		return INT_VALUE
	case "float_value":
		return FLOAT_VALUE
	}
	return COUNT
}

func (s Function) Str() string {
	switch s {
	case COUNT:
		return "count"
	case SUM:
		return "sum"
	case AVG:
		return "avg"
	case MIN:
		return "min"
	case MAX:
		return "max"
	case EXP:
		return "exp"
	case EXTRACT_YEAR:
		return "extract_year"
	case EXTRACT_MONTH:
		return "extract_month"
	case EXTRACT_DAY:
		return "extract_day"
	case EXTRACT_HOUR:
		return "extract_hour"
	case EXTRACT_MINUTE:
		return "extract_minute"
	case EXTRACT_SECOND:
		return "extract_second"
	case INT_VALUE:
		return "int_value"
	case FLOAT_VALUE:
		return "float_value"
	default:
		return fmt.Sprintf("%d", s)
	}
}

type Aggregate struct {
	Function Function `json:"function"`
	Value    Value    `json:"field"`
	As       string   `json:"as"`
}

var (
	aggregateAsPattern = regexp.MustCompile(`^([A-Za-z0-9_]+)\((.+)\):([A-Za-z0-9_]+)$`) // agg(field):as
	aggregatePattern   = regexp.MustCompile(`^([A-Za-z0-9_]+)\((.+)\)$`)                 // agg(field)
)

/**
* ToAggregate: Parses an aggregate expression "agg(field)" or "agg(field):as".
* Without an alias, the alias is the function name (count(id) → as "count").
* @param s string
* @return Aggregate, bool (false when s is not an aggregate or the function is unknown)
**/
func ToAggregate(s string) (Aggregate, bool) {
	s = strings.TrimSpace(s)
	var name, field, as string
	if matches := aggregateAsPattern.FindStringSubmatch(s); matches != nil {
		name, field, as = matches[1], matches[2], matches[3]
	} else if matches := aggregatePattern.FindStringSubmatch(s); matches != nil {
		name, field = matches[1], matches[2]
	} else {
		return Aggregate{}, false
	}

	name = strings.ToLower(name)
	fn := ToFunction(name)
	if fn.Str() != name {
		return Aggregate{}, false
	}

	field = strings.TrimSpace(field)
	if field == "" {
		return Aggregate{}, false
	}

	if as == "" {
		as = name
	}

	return Aggregate{Function: fn, Value: NewValue(field), As: as}, true
}

/**
* Field: Syntactic description of a field reference, independent of any model.
* Source is the name or alias of the origin ("A" in "A.name"); empty means the first origin.
* Agg is nil when the field is not an aggregate.
**/
type Field struct {
	Source string     `json:"source"`
	Name   string     `json:"name"`
	As     string     `json:"as"`
	Agg    *Aggregate `json:"agg,omitempty"`
	Page   int        `json:"page"`
}

var (
	fieldPagePattern = regexp.MustCompile(`^([^|]+)\|page:(\d+)$`)                                          // field|page:n
	fieldPattern     = regexp.MustCompile(`^(?:([A-Za-z0-9_]+)\.)?([A-Za-z0-9_>-]+)(?::([A-Za-z0-9_]+))?$`) // [from.]field[:as]
)

/**
* ToField: Parses a field reference: "field", "field:as", "from.field", "from.field:as",
* "agg(field)", "agg(field):as", "field->a->b" and the "|page:n" suffix.
* Without an alias, As is the field name, or the function name for aggregates.
* @param s string
* @return Field, bool (false when s does not match any form)
**/
func ToField(s string) (Field, bool) {
	s = strings.TrimSpace(s)
	page := 1
	if matches := fieldPagePattern.FindStringSubmatch(s); matches != nil {
		s = strings.TrimSpace(matches[1])
		if n, err := strconv.Atoi(matches[2]); err == nil && n > 0 {
			page = n
		}
	}

	if agg, ok := ToAggregate(s); ok {
		result, ok := ToField(agg.Value.String())
		if !ok || result.Agg != nil {
			return Field{}, false
		}
		result.As = agg.As
		result.Agg = &agg
		result.Page = page
		return result, true
	}

	matches := fieldPattern.FindStringSubmatch(s)
	if matches == nil {
		return Field{}, false
	}

	as := matches[3]
	if as == "" {
		as = matches[2]
	}

	return Field{
		Source: matches[1],
		Name:   matches[2],
		As:     as,
		Page:   page,
	}, true
}

/**
* Exp: Wraps an expression into a Value.
* @param value any
* @return Value
**/
func Exp(value any) Aggregate {
	return Aggregate{Function: EXP, Value: NewValue(value)}
}

/**
* Sum: Returns a SUM aggregate.
* @param value any
* @return Aggregate
**/
func Sum(value any) Aggregate {
	return Aggregate{Function: SUM, Value: NewValue(value)}
}

/**
* Avg: Returns an AVG aggregate.
* @param value any
* @return Aggregate
**/
func Avg(value any) Aggregate {
	return Aggregate{Function: AVG, Value: NewValue(value)}
}

/**
* Min: Returns a MIN aggregate.
* @param value any
* @return Aggregate
**/
func Min(value any) Aggregate {
	return Aggregate{Function: MIN, Value: NewValue(value)}
}

/**
* Max: Returns a MAX aggregate.
* @param value any
* @return Aggregate
**/
func Max(value any) Aggregate {
	return Aggregate{Function: MAX, Value: NewValue(value)}
}

/**
* ExtractYear: Returns a EXTRACT_YEAR aggregate.
* @param value any
* @return Aggregate
**/
func Year(value any) Aggregate {
	return Aggregate{Function: EXTRACT_YEAR, Value: NewValue(value)}
}

/**
* ExtractMonth: Returns a EXTRACT_MONTH aggregate.
* @param value any
* @return Aggregate
**/
func Month(value any) Aggregate {
	return Aggregate{Function: EXTRACT_MONTH, Value: NewValue(value)}
}

/**
* ExtractDay: Returns a EXTRACT_DAY aggregate.
* @param value any
* @return Aggregate
**/
func Day(value any) Aggregate {
	return Aggregate{Function: EXTRACT_DAY, Value: NewValue(value)}
}

/**
* ExtractHour: Returns a EXTRACT_HOUR aggregate.
* @param value any
* @return Aggregate
**/
func Hour(value any) Aggregate {
	return Aggregate{Function: EXTRACT_HOUR, Value: NewValue(value)}
}

/**
* ExtractMinute: Returns a EXTRACT_MINUTE aggregate.
* @param value any
* @return Aggregate
**/
func Minute(value any) Aggregate {
	return Aggregate{Function: EXTRACT_MINUTE, Value: NewValue(value)}
}

/**
* ExtractSecond: Returns a EXTRACT_SECOND aggregate.
* @param value any
* @return Aggregate
**/
func Second(value any) Aggregate {
	return Aggregate{Function: EXTRACT_SECOND, Value: NewValue(value)}
}

/**
* Int: Returns a INT_VALUE aggregate.
* @param value any
* @return Aggregate
**/
func Int(value any) Aggregate {
	return Aggregate{Function: INT_VALUE, Value: NewValue(value)}
}

/**
* Float: Returns a FLOAT_VALUE aggregate.
* @param value any
* @return Aggregate
**/
func Float(value any) Aggregate {
	return Aggregate{Function: FLOAT_VALUE, Value: NewValue(value)}
}

type Connector string

const (
	NaC Connector = ""
	AND Connector = "and"
	OR  Connector = "or"
)

func ToConnector(s string) Connector {
	switch s {
	case "and":
		return AND
	case "or":
		return OR
	}
	return NaC
}

func (s Connector) Str() string {
	return string(s)
}

type Condition struct {
	Field     Value     `json:"field"`
	Operator  Operator  `json:"operator"`
	Value     Value     `json:"value"`
	Connector Connector `json:"connector"`
}

/**
* ToCondition: Converts a JSON object into a Condition.
* @param params Json
* @return *Condition, error
**/
func ToCondition(params Json) (*Condition, error) {
	condition := func(fld string, wr Json) *Condition {
		for k, v := range wr {
			op := ToOperator(k)
			return &Condition{
				Field:     NewValue(fld),
				Operator:  op,
				Value:     NewValue(v),
				Connector: NaC,
			}
		}
		return nil
	}

	fldCondition := func(wr Json) *Condition {
		for k := range wr {
			cmd := wr.Json(k)
			result := condition(k, cmd)
			if result != nil {
				return result
			}
			return result
		}
		return nil
	}

	for key := range params {
		switch strings.ToLower(key) {
		case "and":
			value := params.Json(key)
			result := fldCondition(value)
			if result != nil {
				result.Connector = AND
				return result, nil
			}
		case "or":
			value := params.Json(key)
			result := fldCondition(value)
			if result != nil {
				result.Connector = OR
				return result, nil
			}
		default:
			value := params.Json(key)
			result := condition(key, value)
			if result != nil {
				return result, nil
			}
		}
	}

	return nil, errors.New(MSG_INVALID_CONDITION)
}

/**
* ToConditions: Converts a list of JSON objects into a list of Conditions.
* @param params []Json
* @return []*Condition, error
**/
func ToConditions(params []Json) ([]*Condition, error) {
	result := make([]*Condition, 0)
	for _, param := range params {
		condition, err := ToCondition(param)
		if err != nil {
			return nil, err
		}
		result = append(result, condition)
	}
	return result, nil
}

/**
* ToJson
* @return Json
**/
func (s *Condition) ToJson() Json {
	if s.Connector == NaC {
		return Json{
			s.Field.String(): Json{
				s.Operator.Str(): s.Value.Value,
			},
		}
	}

	return Json{
		s.Connector.Str(): Json{
			s.Field.String(): Json{
				s.Operator.Str(): s.Value.Value,
			},
		},
	}
}

/**
* And
* @return *Condition
**/
func (s *Condition) And() *Condition {
	s.Connector = AND
	return s
}

/**
* Or
* @return *Condition
**/
func (s *Condition) Or() *Condition {
	s.Connector = OR
	return s
}

/**
* fieldValue
* @param data Json
* @return any, error
**/
func (s *Condition) fieldValue(data Json) (result any) {
	result = data.Clone()
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
		if len(bv) == 0 {
			return invalidType()
		}
		value, ok := firstMapValueSorted(bv[0])
		if !ok {
			return invalidType()
		}
		tmp := *s
		tmp.Value = NewValue(value)
		return tmp.applyOpLess(val)
	case []interface{}:
		for _, value := range bv {
			tmp := *s
			tmp.Value = NewValue(value)
			return tmp.applyOpLess(val)
		}
		return invalidType()
	default:
		if first, ok := firstOfSlice(bv); ok {
			tmp := *s
			tmp.Value = NewValue(first)
			return tmp.applyOpLess(val)
		}
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
		if len(bv) == 0 {
			return invalidType()
		}
		value, ok := firstMapValueSorted(bv[0])
		if !ok {
			return invalidType()
		}
		tmp := *s
		tmp.Value = NewValue(value)
		return tmp.applyOpLessEq(val)
	case []interface{}:
		for _, value := range bv {
			tmp := *s
			tmp.Value = NewValue(value)
			return tmp.applyOpLessEq(val)
		}
		return invalidType()
	default:
		if first, ok := firstOfSlice(bv); ok {
			tmp := *s
			tmp.Value = NewValue(first)
			return tmp.applyOpLessEq(val)
		}
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
		if len(bv) == 0 {
			return invalidType()
		}
		value, ok := firstMapValueSorted(bv[0])
		if !ok {
			return invalidType()
		}
		tmp := *s
		tmp.Value = NewValue(value)
		return tmp.applyOpMore(val)
	case []interface{}:
		for _, value := range bv {
			tmp := *s
			tmp.Value = NewValue(value)
			return tmp.applyOpMore(val)
		}
		return invalidType()
	default:
		if first, ok := firstOfSlice(bv); ok {
			tmp := *s
			tmp.Value = NewValue(first)
			return tmp.applyOpMore(val)
		}
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
		if len(bv) == 0 {
			return invalidType()
		}
		value, ok := firstMapValueSorted(bv[0])
		if !ok {
			return invalidType()
		}
		tmp := *s
		tmp.Value = NewValue(value)
		return tmp.applyOpMoreEq(val)
	case []interface{}:
		for _, value := range bv {
			tmp := *s
			tmp.Value = NewValue(value)
			return tmp.applyOpMoreEq(val)
		}
		return invalidType()
	default:
		if first, ok := firstOfSlice(bv); ok {
			tmp := *s
			tmp.Value = NewValue(first)
			return tmp.applyOpMoreEq(val)
		}
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
* condition
* @param field, value interface{}, op string
* @return *Condition
**/
func condition(field, value interface{}, op Operator) *Condition {
	return &Condition{
		Field:     NewValue(field),
		Operator:  op,
		Value:     NewValue(value),
		Connector: NaC,
	}
}

/**
* Where
* @param field, value interface{}
* @return Condition
**/
func Where(field, value interface{}) *Condition {
	return condition(field, value, EQ)
}

/**
* And
* @param field, operator Operator, value interface{}
* @return Condition
**/
func And(field interface{}, operator Operator, value interface{}) *Condition {
	result := condition(field, value, operator)
	result.Connector = AND
	return result
}

/**
* Or
* @param field, operator Operator, value interface{}
* @return Condition
**/
func Or(field interface{}, operator Operator, value interface{}) *Condition {
	result := condition(field, value, operator)
	result.Connector = OR
	return result
}

/**
* Eq
* @param field, value interface{}
* @return Condition
**/
func Eq(field, value interface{}) *Condition {
	return condition(field, value, EQ)
}

/**
* Neg
* @param field, value interface{}
* @return Condition
**/
func Neg(field, value interface{}) *Condition {
	return condition(field, value, NEG)
}

/**
* Less
* @param field, value interface{}
* @return Condition
**/
func Less(field, value interface{}) *Condition {
	return condition(field, value, LESS)
}

/**
* LessEq
* @param field, value interface{}
* @return Condition
**/
func LessEq(field, value interface{}) *Condition {
	return condition(field, value, LESS_EQ)
}

/**
* More
* @param field, value interface{}
* @return Condition
**/
func More(field, value interface{}) *Condition {
	return condition(field, value, MORE)
}

/**
* MoreEq
* @param field, value interface{}
* @return Condition
**/
func MoreEq(field, value interface{}) *Condition {
	return condition(field, value, MORE_EQ)
}

/**
* Like
* @param field, value interface{}
* @return Condition
**/
func Like(field, value interface{}) *Condition {
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
* @param field, value interface{}
* @return Condition
**/
func Is(field, value interface{}) *Condition {
	return condition(field, value, IS)
}

/**
* IsNot
* @param field, value interface{}
* @return Condition
**/
func IsNot(field, value interface{}) *Condition {
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
* EvaluateObject: Evalúa las condiciones sobre un objeto, encadenándolas por su conector (AND / OR).
* @param obj Json, conditions []*Condition
* @return bool
**/
func EvaluateObject(obj Json, conditions []*Condition) bool {
	if len(conditions) == 0 {
		return true
	}

	result := conditions[0].ApplyToObject(obj)
	for _, cond := range conditions[1:] {
		ok := cond.ApplyToObject(obj)
		if cond.Connector == AND {
			result = result && ok
		} else if cond.Connector == OR {
			result = result || ok
		}
	}

	return result
}

/**
* EvaluateValue
* @param value any, conditions []*Condition
* @return bool
**/
func EvaluateValue(value any, conditions []*Condition) bool {
	if len(conditions) == 0 {
		return true
	}

	result := conditions[0].ApplyToValue(value)
	for _, cond := range conditions[1:] {
		ok := cond.ApplyToValue(value)
		if cond.Connector == AND {
			result = result && ok
		} else if cond.Connector == OR {
			result = result || ok
		}
	}

	return result
}

type WheRes []*Condition

/**
* ToWheRes
* @param params []Json
* @return WheRes, error
**/
func ToWheRes(params []Json) (WheRes, error) {
	result := WheRes{}
	for _, param := range params {
		condition, err := ToCondition(param)
		if err != nil {
			return nil, err
		}
		result = append(result, condition)
	}
	return result, nil
}

/**
* Len
* @return int
**/
func (s WheRes) Len() int {
	return len(s)
}

/**
* Add
* @param condition *Condition
**/
func (s WheRes) Add(condition *Condition) {
	s = append(s, condition)
}

/**
* ToJson
* @return []Json
**/
func (s WheRes) ToJson() []Json {
	result := make([]Json, 0)
	for _, condition := range s {
		cond := condition.ToJson()
		result = append(result, cond)
	}
	return result
}
