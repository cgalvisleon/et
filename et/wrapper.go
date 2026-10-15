package et

/**
* EvaluateObjectWrapper: EvaluateObject as a script (goja, jrex/jwf) sees it: evaluateObject(obj, where) is true when
* obj meets the conditions. where is a list of conditions as JSON, as in a jsql where; an invalid condition comes back
* as a Go error, which goja throws as a JavaScript Error:
*
*	evaluateObject(params, [{status: {eq: "active"}}, {and: {age: {more: 18}}}, {or: {role: {eq: "admin"}}}])
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
* EvaluateValueWrapper: EvaluateValue as a script (goja, jrex/jwf) sees it: evaluateValue(value, where) is true when
* value meets the conditions. where is a list of conditions as JSON, as in a jsql where; their field name is not used.
* An invalid condition is thrown as an Error:
*
*	evaluateValue(age, [{value: {more_eq: 18}}, {and: {value: {less: 65}}}])
*
* @return func(value any, where []Json) (bool, error)
**/
func EvaluateValueWrapper() func(value any, where []Json) (bool, error) {
	return func(value any, where []Json) (bool, error) {
		conditions, err := toConditionList(where)
		if err != nil {
			return false, err
		}
		return EvaluateValue(value, conditions), nil
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
