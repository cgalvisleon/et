package et

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrorFieldNotFound = errors.New(MSG_FIELD_NOT_FOUND)
	ErrorDataNotFound  = errors.New(MSG_DATA_NOT_FOUND)
)

/**
* Operator: Comparison operator of a Condition, as written in JSON ("eq", "less_eq", "not_between"...).
* Adding one: the constants, ToOperator and a constructor here (condition.go); ApplyToValue in evaluate.go;
* et/spec.md §3.2; the five jsql/drivers/<x>/query.go; the constructor in jsql/catalog.go and jsql/wrapper.go;
* jsql/jsql.d.ts (JsqlOperator); the tests condition_test.go and jsql/drivers/test/evaluate_test.go.
**/
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

/**
* Str: The operator as text ("eq").
* @return string
**/
func (s Operator) Str() string {
	return string(s)
}

/**
* ToOperator: Reads an operator from its text, case insensitive; an unknown one is EQ.
* @param s string
* @return Operator
**/
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
* Time: Converts a time.Time, *time.Time or RFC3339 text to a time.
* @param val any
* @return *time.Time (nil when it cannot be converted)
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

/**
* TypeData: Logical type of a Value or a column (TEXT, INT, ARRAY_JSON...); valueType infers it from a Go value.
* Adding one: the constants, TypeValues and valueType here (condition.go); the five jsql/drivers/<x>/types.go and
* quoted.go (sqlite also condition.go); jsql/jsql.d.ts (JsqlTypeData).
**/
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
	FIELD          TypeData = "field"
)

/**
* Str: The type as text ("int").
* @return string
**/
func (s TypeData) Str() string {
	return string(s)
}

/**
* TypeValues: Every TypeData by its text, for IsTypeValue and Value.Is; FIELD is not listed.
**/
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

/**
* IsTypeValue: Whether v (case insensitive) is the text of a TypeData in TypeValues.
* @param v string
* @return bool
**/
func IsTypeValue(v string) bool {
	v = strings.ToLower(v)
	_, exists := TypeValues[v]
	return exists
}

/**
* Value: A raw value with its logical Type; the field and the value of a Condition are both a Value.
**/
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
	case *Field:
		if t != nil {
			return Value{Type: FIELD, Value: *t}
		}
		return Value{Type: valueType(nil), Value: nil}
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
	switch val := v.(type) {
	case nil:
		return VAL_NULL
	case string:
		if len(val) > 255 {
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
	case Field:
		return FIELD
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

/**
* BetweenValue: The inclusive range of a between/not_between condition (ToBetween reads it from JSON).
**/
type BetweenValue struct {
	Min any `json:"Min"`
	Max any `json:"Max"`
}

/**
* Function: Aggregate or scalar function of an Aggregate (count, sum, extract_year, int_value...), by position.
* Adding one: the constants, ToFunction, Str and a constructor here (condition.go); applyFunction in evaluate.go;
* FieldWrapper in goja.go; et/spec.md §3.6; the five jsql/drivers/<x>/quoted.go; jsql/jsql.d.ts (declare function);
* the test evaluate_test.go.
**/
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

/**
* ToFunction: Reads a function from its lowercase name; an unknown one is COUNT (ToAggregate detects it with Str).
* @param s string
* @return Function
**/
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

/**
* Str: The function name ("sum"); an unknown value gives its number.
* @return string
**/
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

/**
* Aggregate: A Function applied to a Value (a field or an expression), with an optional alias As.
**/
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
* Fld: Returns a Field.
* @param name string
* @return Field
**/
func Fld(name string) Field {
	result, ok := ToField(name)
	if !ok {
		return Field{
			Name: name,
			As:   name,
		}
	}
	return result
}

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
* Exp: Returns an EXP aggregate over a raw expression.
* @param value any
* @return Aggregate
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
* Year: Returns an EXTRACT_YEAR aggregate.
* @param value any
* @return Aggregate
**/
func Year(value any) Aggregate {
	return Aggregate{Function: EXTRACT_YEAR, Value: NewValue(value)}
}

/**
* Month: Returns an EXTRACT_MONTH aggregate.
* @param value any
* @return Aggregate
**/
func Month(value any) Aggregate {
	return Aggregate{Function: EXTRACT_MONTH, Value: NewValue(value)}
}

/**
* Day: Returns an EXTRACT_DAY aggregate.
* @param value any
* @return Aggregate
**/
func Day(value any) Aggregate {
	return Aggregate{Function: EXTRACT_DAY, Value: NewValue(value)}
}

/**
* Hour: Returns an EXTRACT_HOUR aggregate.
* @param value any
* @return Aggregate
**/
func Hour(value any) Aggregate {
	return Aggregate{Function: EXTRACT_HOUR, Value: NewValue(value)}
}

/**
* Minute: Returns an EXTRACT_MINUTE aggregate.
* @param value any
* @return Aggregate
**/
func Minute(value any) Aggregate {
	return Aggregate{Function: EXTRACT_MINUTE, Value: NewValue(value)}
}

/**
* Second: Returns an EXTRACT_SECOND aggregate.
* @param value any
* @return Aggregate
**/
func Second(value any) Aggregate {
	return Aggregate{Function: EXTRACT_SECOND, Value: NewValue(value)}
}

/**
* Int: Returns an INT_VALUE aggregate.
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

/**
* Connector: How a Condition joins the previous one: AND, OR or NaC (none, the first of a list).
**/
type Connector string

const (
	NaC Connector = ""
	AND Connector = "and"
	OR  Connector = "or"
)

/**
* ToConnector: Reads a connector from "and"/"or"; anything else is NaC.
* @param s string
* @return Connector
**/
func ToConnector(s string) Connector {
	switch s {
	case "and":
		return AND
	case "or":
		return OR
	}
	return NaC
}

/**
* Str: The connector as text ("and"); empty for NaC.
* @return string
**/
func (s Connector) Str() string {
	return string(s)
}

/**
* Condition: One comparison, Field Operator Value, joined to the previous condition by Connector.
* ToCondition reads it from {field: {op: value}} or {where|and|or: {field: {op: value}}}.
**/
type Condition struct {
	Field     Value     `json:"field"`
	Operator  Operator  `json:"operator"`
	Value     Value     `json:"value"`
	Connector Connector `json:"connector"`
}

/**
* Command: Kind of a write command: insert, update, delete or upsert.
**/
type Command string

const (
	INSERT Command = "insert"
	UPDATE Command = "update"
	DELETE Command = "delete"
	UPSERT Command = "upsert"
)

/**
* ToCondition: Converts a JSON object into a Condition.
* @param params Json
* @return *Condition, error
**/
func ToCondition(params Json) (*Condition, error) {
	conditionOf := func(fld string, wr Json) *Condition {
		for k, v := range wr {
			op := ToOperator(k)
			value := toConditionValue(v)
			if op == BETWEEN || op == NOT_BETWEEN {
				if bv, ok := ToBetween(v); ok {
					value = NewValue(bv)
				}
			}
			return &Condition{
				Field:     NewValue(fld),
				Operator:  op,
				Value:     value,
				Connector: NaC,
			}
		}
		return nil
	}

	firstConditionOf := func(wr Json) *Condition {
		for k := range wr {
			return conditionOf(k, wr.Json(k))
		}
		return nil
	}

	for key := range params {
		switch strings.ToLower(key) {
		case "where":
			// The first condition of a list: {"where": {field: {operator: value}}}, without a connector
			value := params.Json(key)
			result := firstConditionOf(value)
			if result != nil {
				return result, nil
			}
		case "and":
			value := params.Json(key)
			result := firstConditionOf(value)
			if result != nil {
				result.Connector = AND
				return result, nil
			}
		case "or":
			value := params.Json(key)
			result := firstConditionOf(value)
			if result != nil {
				result.Connector = OR
				return result, nil
			}
		default:
			value := params.Json(key)
			result := conditionOf(key, value)
			if result != nil {
				return result, nil
			}
		}
	}

	return nil, errors.New(MSG_INVALID_CONDITION)
}

/**
* ToConditions: Converts a list of JSON objects into a list of Conditions; one without connector after the first is an AND (AddCondition).
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
		result = AddCondition(result, condition)
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
				s.Operator.Str(): jsonValue(s.Value),
			},
		}
	}

	return Json{
		s.Connector.Str(): Json{
			s.Field.String(): Json{
				s.Operator.Str(): jsonValue(s.Value),
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
* WheRes: A list of conditions read from JSON (ToWheRes). Unlike ToConditions it keeps the connectors as they
* came: a missing one after the first is not turned into AND (the jsql callers do it).
**/
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
func (s *WheRes) Add(condition *Condition) {
	*s = append(*s, condition)
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

/**
* AddCondition: Appends a condition to the conditions list.
* @param conditions []*Condition, cond *Condition
* @return []*Condition
**/
func AddCondition(conditions []*Condition, cond *Condition) []*Condition {
	if cond == nil {
		return conditions
	}
	if conditions == nil {
		conditions = make([]*Condition, 0)
	}
	if len(conditions) > 0 && cond.Connector == NaC {
		cond.Connector = AND
	}
	return append(conditions, cond)
}

/**
* ToWhere
* @param cond any
* @return []Json
**/
func ToWhere(cond any) []Json {
	switch value := cond.(type) {
	case *Condition:
		if value == nil {
			return nil
		}
		return []Json{value.ToJson()}
	case []*Condition:
		result := make([]Json, 0, len(value))
		for _, c := range value {
			result = append(result, ToWhere(c)...)
		}
		return result
	case WheRes:
		return value.ToJson()
	case []Json:
		return value
	case []any:
		result := make([]Json, 0, len(value))
		for _, item := range value {
			result = append(result, ToWhere(item)...)
		}
		return result
	}
	obj, ok := asJson(cond)
	if !ok {
		return nil
	}
	// One object is one condition per key ({field: {op: value}} or {where|and|or: {...}}),
	// with where first: the field names are kept (B14)
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		wi, wj := strings.EqualFold(keys[i], "where"), strings.EqualFold(keys[j], "where")
		if wi != wj {
			return wi
		}
		return keys[i] < keys[j]
	})
	result := make([]Json, 0, len(keys))
	for _, k := range keys {
		result = append(result, Json{k: obj[k]})
	}
	return result
}
