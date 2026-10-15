package et

import (
	"fmt"
	"strings"
)

/**
* EvaluateObjectWrapper: EvaluateObject as a script (goja, jrex/jwf) sees it: evaluateObject(obj, where) is true when
* obj meets the conditions. where is a list of conditions as JSON, as in a jsql where; an invalid condition comes back
* as a Go error, which goja throws as a JavaScript Error:
*
*	evaluateObject(params, [{status: {eq: "active"}}, {and: {age: {more: 18}}}, {or: {role: {eq: "admin"}}}])
*	evaluateObject(params, [{age: {more_eq: field("min_age")}}, {"extract_year(created_at)": {eq: 2026}}])
*
* @return func(obj Json, where []Json) (bool, error)
**/
func EvaluateObjectWrapper() func(obj Json, where []Json) (bool, error) {
	return func(obj Json, where []Json) (bool, error) {
		conditions, err := toConditionList(where)
		if err != nil {
			return false, err
		}
		return EvaluateObject(obj, conditions), nil
	}
}

/**
* EvaluateValueWrapper: EvaluateValue as a script (goja, jrex/jwf) sees it: evaluateValue(value, where, obj?) is true
* when value meets the conditions. where is a list of conditions as JSON, as in a jsql where; their field name is not
* used. A field reference ({field: "min_age"}, field("min_age"), sum("items")…) is read from obj; without obj it does
* not hold. An invalid condition is thrown as an Error:
*
*	evaluateValue(age, [{value: {more_eq: 18}}, {and: {value: {less: 65}}}])
*	evaluateValue(age, [{value: {more_eq: field("min_age")}}], params)
*
* @return func(value any, where []Json, obj ...Json) (bool, error)
**/
func EvaluateValueWrapper() func(value any, where []Json, obj ...Json) (bool, error) {
	return func(value any, where []Json, obj ...Json) (bool, error) {
		conditions, err := toConditionList(where)
		if err != nil {
			return false, err
		}
		if len(obj) > 0 && obj[0] != nil {
			for i, cond := range conditions {
				conditions[i] = cond.resolve(obj[0])
			}
		}
		return EvaluateValue(value, conditions), nil
	}
}

/**
* FieldWrapper: The field references of a script (goja, jrex/jwf), to set as globals: field(name) and the aggregates
* count, sum, avg, min, max, year, month, day, hour, minute, second, intValue and floatValue. Each gives the JSON of a
* reference, {field: "sum(items)"}, so it can go in a condition, be stored and be printed:
*
*	evaluateObject(params, [{age: {more_eq: field("min_age")}}, {and: {total: {eq: sum("items->price")}}}])
*
* @return map[string]any
**/
func FieldWrapper() map[string]any {
	ref := func(fn Function) func(name string) map[string]any {
		return func(name string) map[string]any {
			return map[string]any{FIELD.Str(): fmt.Sprintf("%s(%s)", fn.Str(), strings.TrimSpace(name))}
		}
	}
	return map[string]any{
		"field": func(name string) map[string]any {
			return map[string]any{FIELD.Str(): strings.TrimSpace(name)}
		},
		"count":      ref(COUNT),
		"sum":        ref(SUM),
		"avg":        ref(AVG),
		"min":        ref(MIN),
		"max":        ref(MAX),
		"year":       ref(EXTRACT_YEAR),
		"month":      ref(EXTRACT_MONTH),
		"day":        ref(EXTRACT_DAY),
		"hour":       ref(EXTRACT_HOUR),
		"minute":     ref(EXTRACT_MINUTE),
		"second":     ref(EXTRACT_SECOND),
		"intValue":   ref(INT_VALUE),
		"floatValue": ref(FLOAT_VALUE),
	}
}

/**
* toConditionList: The conditions of a list of JSON conditions; one without a connector after the first goes with AND.
* @param where []Json
* @return []*Condition, error
**/
func toConditionList(where []Json) ([]*Condition, error) {
	conditions, err := ToConditions(where)
	if err != nil {
		return nil, err
	}
	result := make([]*Condition, 0, len(conditions))
	for _, cond := range conditions {
		result = AddCondition(result, cond)
	}
	return result, nil
}
