package validator

import (
	"fmt"
	"regexp"

	"github.com/cgalvisleon/et/et"
)

/**
* Wrapper: The validator as a script (goja, jrex/jwf) sees it: validator(def?) gives a new validator, from the json
* definition of ByJson when it comes. Its methods are camelCase and chain; every error (an invalid definition or data
* that does not pass) comes back as a Go error, which goja throws as a JavaScript Error, so the script catches it with
* try/catch (e.message):
*
*	try {
*	  validator().field("email").required().isEmail().field("age").min(18).validate(params)
*	  validator({fields: {email: {required: true, is_email: true}}}).validate(params)
*	} catch (e) { ctx.set({error: e.message}) }
*
* @return func(def ...et.Json) (map[string]any, error)
**/
func Wrapper() func(def ...et.Json) (map[string]any, error) {
	return func(def ...et.Json) (map[string]any, error) {
		result := New()
		if len(def) > 0 {
			if _, err := result.ByJson(def[0]); err != nil {
				return nil, err
			}
		}
		return wrapValidator(result), nil
	}
}

/**
* wrapValidator: A *Validator for a script: notEmpty, field, byJson, validate (true, or throws why not), isValid (true
* or false, never throws), errorOf (why not, or ""), toJson
* @param s *Validator
* @return map[string]any
**/
func wrapValidator(s *Validator) map[string]any {
	var result map[string]any
	result = map[string]any{
		"notEmpty": func() map[string]any {
			s.NotEmpty()
			return result
		},
		"field": func(name string) map[string]any {
			return wrapCondition(s.Field(name))
		},
		"byJson": func(def et.Json) (map[string]any, error) {
			if _, err := s.ByJson(def); err != nil {
				return nil, err
			}
			return result, nil
		},
		"validate": s.Validate,
		"isValid": func(data et.Json) bool {
			ok, err := s.Validate(data)
			return ok && err == nil
		},
		"errorOf": func(data et.Json) string {
			return errorOf(s.Validate(data))
		},
		"toJson": func() map[string]any {
			return s.ToJson()
		},
	}
	return result
}

/**
* wrapCondition: A *Condition for a script: its rules chain (required().isEmail().maxLength(120)), field goes on with
* another field and validate/isValid/errorOf check the data with the whole validator. pattern throws if it does not
* compile (it would make validate panic)
* @param s *Condition
* @return map[string]any
**/
func wrapCondition(s *Condition) map[string]any {
	var result map[string]any
	rule := func(apply func()) func() map[string]any {
		return func() map[string]any {
			apply()
			return result
		}
	}
	number := func(apply func(value float64)) func(value float64) map[string]any {
		return func(value float64) map[string]any {
			apply(value)
			return result
		}
	}
	length := func(apply func(value int)) func(value int) map[string]any {
		return func(value int) map[string]any {
			apply(value)
			return result
		}
	}
	result = map[string]any{
		"required":            rule(func() { s.Required() }),
		"notEmpty":            rule(func() { s.NotEmpty() }),
		"isLetters":           rule(func() { s.IsLetters() }),
		"isNumbers":           rule(func() { s.IsNumbers() }),
		"isSpecialCharacters": rule(func() { s.IsSpecialCharacters() }),
		"isEmail":             rule(func() { s.IsEmail() }),
		"isPhone":             rule(func() { s.IsPhone() }),
		"min":                 number(func(value float64) { s.Min(value) }),
		"max":                 number(func(value float64) { s.Max(value) }),
		"minLength":           length(func(value int) { s.MinLength(value) }),
		"maxLength":           length(func(value int) { s.MaxLength(value) }),
		"between": func(min, max float64) map[string]any {
			s.Between(min, max)
			return result
		},
		"pattern": func(pattern string) (map[string]any, error) {
			if _, err := regexp.Compile(pattern); err != nil {
				return nil, fmt.Errorf(MSG_VALIDATOR_INVALID_PATTERN, s.name, pattern)
			}
			s.Pattern(pattern)
			return result, nil
		},
		"values": func(values ...any) map[string]any {
			// values("a", "b") o values(["a", "b"])
			if len(values) == 1 {
				if list, ok := values[0].([]any); ok {
					values = list
				}
			}
			s.Values(values...)
			return result
		},
		"field": func(name string) map[string]any {
			return wrapCondition(s.Field(name))
		},
		"validate": s.Validate,
		"isValid": func(data et.Json) bool {
			ok, err := s.Validate(data)
			return ok && err == nil
		},
		"errorOf": func(data et.Json) string {
			return errorOf(s.Validate(data))
		},
		"toJson": func() map[string]any {
			return s.ToJson()
		},
	}
	return result
}

/**
* errorOf: Why a validation did not pass, or "" when it did
* @param ok bool, err error
* @return string
**/
func errorOf(ok bool, err error) string {
	if err != nil {
		return err.Error()
	}
	if !ok {
		return MSG_VALIDATOR_EMPTY
	}
	return ""
}
