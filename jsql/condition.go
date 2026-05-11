package jsql

import "github.com/cgalvisleon/et/et"

/**
* Eq: Returns an equality condition (field = value).
* @param field string
* @param value interface{}
* @return *et.Condition
**/
func Eq(field string, value interface{}) *et.Condition {
	return et.Eq(field, value)
}

/**
* Neg: Returns a not-equal condition (field <> value).
* @param field string
* @param value interface{}
* @return *et.Condition
**/
func Neg(field string, value interface{}) *et.Condition {
	return et.Neg(field, value)
}

/**
* Less: Returns a less-than condition (field < value).
* @param field string
* @param value interface{}
* @return *et.Condition
**/
func Less(field string, value interface{}) *et.Condition {
	return et.Less(field, value)
}

/**
* LessEq: Returns a less-than-or-equal condition (field <= value).
* @param field string
* @param value interface{}
* @return *et.Condition
**/
func LessEq(field string, value interface{}) *et.Condition {
	return et.LessEq(field, value)
}

/**
* More: Returns a greater-than condition (field > value).
* @param field string
* @param value interface{}
* @return *et.Condition
**/
func More(field string, value interface{}) *et.Condition {
	return et.More(field, value)
}

/**
* MoreEq: Returns a greater-than-or-equal condition (field >= value).
* @param field string
* @param value interface{}
* @return *et.Condition
**/
func MoreEq(field string, value interface{}) *et.Condition {
	return et.MoreEq(field, value)
}

/**
* Like: Returns a case-insensitive pattern match condition (field ILIKE value).
* @param field string
* @param value interface{}
* @return *et.Condition
**/
func Like(field string, value interface{}) *et.Condition {
	return et.Like(field, value)
}

/**
* In: Returns an inclusion condition (field IN (values...)).
* @param field string
* @param value []interface{}
* @return *et.Condition
**/
func In(field string, value []interface{}) *et.Condition {
	return et.In(field, value)
}

/**
* NotIn: Returns an exclusion condition (field NOT IN (values...)).
* @param field string
* @param value []interface{}
* @return *et.Condition
**/
func NotIn(field string, value []interface{}) *et.Condition {
	return et.NotIn(field, value)
}

/**
* Is: Returns an IS condition (field IS value), typically used with NULL or booleans.
* @param field string
* @param value interface{}
* @return *et.Condition
**/
func Is(field string, value interface{}) *et.Condition {
	return et.Is(field, value)
}

/**
* IsNot: Returns an IS NOT condition (field IS NOT value).
* @param field string
* @param value interface{}
* @return *et.Condition
**/
func IsNot(field string, value interface{}) *et.Condition {
	return et.IsNot(field, value)
}

/**
* Null: Returns an IS NULL condition (field IS NULL).
* @param field string
* @return *et.Condition
**/
func Null(field string) *et.Condition {
	return et.Null(field)
}

/**
* NotNull: Returns an IS NOT NULL condition (field IS NOT NULL).
* @param field string
* @return *et.Condition
**/
func NotNull(field string) *et.Condition {
	return et.NotNull(field)
}

/**
* Between: Returns a range condition (field BETWEEN min AND max).
* @param field string
* @param min any
* @param max any
* @return *et.Condition
**/
func Between(field string, min, max any) *et.Condition {
	return et.Between(field, min, max)
}

/**
* NotBetween: Returns a negated range condition (field NOT BETWEEN min AND max).
* @param field string
* @param min any
* @param max any
* @return *et.Condition
**/
func NotBetween(field string, min, max any) *et.Condition {
	return et.NotBetween(field, min, max)
}

/**
* Evaluate: Returns true if all conditions in the slice match the given JSON object.
* @param item et.Json
* @param conditions []*et.Condition
* @return bool
**/
func Evaluate(item et.Json, conditions []*et.Condition) bool {
	return et.Evaluate(item, conditions)
}
