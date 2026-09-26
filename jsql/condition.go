package jsql

import "github.com/cgalvisleon/et/et"

/**
* where: Returns a WHERE condition (field = value).
* @param field, value interface{}
* @return *et.Condition
**/
func where(field, value interface{}) *et.Condition {
	return et.Where(field, value)
}

/**
* and: Returns a AND condition (field = value).
* @param field, operator et.Operator, value interface{}
* @return *et.Condition
**/
func and(field interface{}, operator et.Operator, value interface{}) *et.Condition {
	return et.And(field, operator, value)
}

/**
* or: Returns a OR condition (field = value).
* @param field, operator et.Operator, value interface{}
* @return *et.Condition
**/
func or(field interface{}, operator et.Operator, value interface{}) *et.Condition {
	return et.Or(field, operator, value)
}

/**
* eq: Returns an equality condition (field = value).
* @param field string, value interface{}
* @return *et.Condition
**/
func eq(field string, value interface{}) *et.Condition {
	return et.Eq(field, value)
}

/**
* neg: Returns a not-equal condition (field <> value).
* @param field string, value interface{}
* @return *et.Condition
**/
func neg(field string, value interface{}) *et.Condition {
	return et.Neg(field, value)
}

/**
* less: Returns a less-than condition (field < value).
* @param field string, value interface{}
* @return *et.Condition
**/
func less(field string, value interface{}) *et.Condition {
	return et.Less(field, value)
}

/**
* lessEq: Returns a less-than-or-equal condition (field <= value).
* @param field string, value interface{}
* @return *et.Condition
**/
func lessEq(field string, value interface{}) *et.Condition {
	return et.LessEq(field, value)
}

/**
* more: Returns a greater-than condition (field > value).
* @param field string, value interface{}
* @return *et.Condition
**/
func more(field string, value interface{}) *et.Condition {
	return et.More(field, value)
}

/**
* moreEq: Returns a greater-than-or-equal condition (field >= value).
* @param field string, value interface{}
* @return *et.Condition
**/
func moreEq(field string, value interface{}) *et.Condition {
	return et.MoreEq(field, value)
}

/**
* like: Returns a case-insensitive pattern match condition (field ILIKE value).
* @param field string, value interface{}
* @return *et.Condition
**/
func like(field string, value interface{}) *et.Condition {
	return et.Like(field, value)
}

/**
* in: Returns an inclusion condition (field IN (values...)).
* @param field string, value []interface{}
* @return *et.Condition
**/
func in(field string, value []interface{}) *et.Condition {
	return et.In(field, value)
}

/**
* notIn: Returns an exclusion condition (field NOT IN (values...)).
* @param field string, value []interface{}
* @return *et.Condition
**/
func notIn(field string, value []interface{}) *et.Condition {
	return et.NotIn(field, value)
}

/**
* is: Returns an IS condition (field IS value), typically used with NULL or booleans.
* @param field string, value interface{}
* @return *et.Condition
**/
func is(field string, value interface{}) *et.Condition {
	return et.Is(field, value)
}

/**
* isNot: Returns an IS NOT condition (field IS NOT value).
* @param field string, value interface{}
* @return *et.Condition
**/
func isNot(field string, value interface{}) *et.Condition {
	return et.IsNot(field, value)
}

/**
* null: Returns an IS NULL condition (field IS NULL).
* @param field string
* @return *et.Condition
**/
func null(field string) *et.Condition {
	return et.Null(field)
}

/**
* notNull: Returns an IS NOT NULL condition (field IS NOT NULL).
* @param field string
* @return *et.Condition
**/
func notNull(field string) *et.Condition {
	return et.NotNull(field)
}

/**
* between: Returns a range condition (field BETWEEN min AND max).
* @param field string, min any, max any
* @return *et.Condition
**/
func between(field string, min, max any) *et.Condition {
	return et.Between(field, min, max)
}

/**
* notBetween: Returns a negated range condition (field NOT BETWEEN min AND max).
* @param field string, min any, max any
* @return *et.Condition
**/
func notBetween(field string, min, max any) *et.Condition {
	return et.NotBetween(field, min, max)
}
