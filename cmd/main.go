package main

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

func main() {
	query := []et.Json{
		{"id": et.Json{"eq": 1}},
		{"and": et.Json{"name": et.Json{"eq": "Carlos"}}},
	}

	_wherr := QueryWhere(query)

	logs.Debugf("Where: %s", _wherr)
}

func QueryWhere(query []et.Json) string {
	result := ""

	condition := func(col string, value interface{}) string {
		_condition, ok := value.(et.Json)
		if !ok {
			return ""
		}

		for op, v := range _condition {
			val := utility.Unquote(v)
			switch op {
			case "eq":
				return strs.Format(`%s = %v`, col, val)
			case "ne":
				return strs.Format(`%s <> %v`, col, val)
			case "gt":
				return strs.Format(`%s > %v`, col, val)
			case "lt":
				return strs.Format(`%s < %v`, col, val)
			case "gte":
				return strs.Format(`%s >= %v`, col, val)
			case "lte":
				return strs.Format(`%s <= %v`, col, val)
			case "in":
				return strs.Format(`%s IN (%v)`, col, val)
			case "nin":
				return strs.Format(`%s NOT IN (%v)`, col, val)
			case "like":
				return strs.Format(`%s LIKE %v`, col, val)
			case "nlike":
				return strs.Format(`%s NOT LIKE %v`, col, val)
			case "is":
				return strs.Format(`%s IS %v`, col, val)
			case "nis":
				return strs.Format(`%s IS NOT %v`, col, val)
			}
		}

		return ""
	}

	logicCondition := func(operator string, value interface{}) string {
		_value, ok := value.(et.Json)
		if !ok {
			return ""
		}

		_condition := ""
		for key, val := range _value {
			_condition = condition(key, val)
		}

		switch strs.Lowcase(operator) {
		case "and":
			return strs.Format(`AND %s`, _condition)
		case "or":
			return strs.Format(`OR %s`, _condition)
		case "not":
			return strs.Format(`NOT %s`, _condition)
		}

		return ""
	}

	for i, item := range query {
		if i == 0 {
			for key, value := range item {
				result = condition(key, value)
			}
		} else {
			for key, value := range item {
				def := logicCondition(key, value)
				result = strs.Append(result, def, " ")
			}
		}
	}

	return result
}
