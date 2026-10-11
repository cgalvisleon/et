package validator

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/cgalvisleon/et/et"
)

// conditionKeys: the keys a field's condition accepts in ByJson (the Condition methods in snake_case)
var conditionKeys = []string{"required", "not_empty", "is_letters", "is_numbers", "is_special_characters", "is_email", "is_phone",
	"min", "max", "min_length", "max_length", "pattern"}

/**
* ByJson: Define the validator from a json: {not_empty, fields: {name: {required, not_empty, is_email, min, max, min_length, max_length, pattern…}}}
* @param params et.Json
* @return *Validator, error
**/
func (s *Validator) ByJson(params et.Json) (*Validator, error) {
	if params.Bool("not_empty") {
		s.NotEmpty()
	}

	fields := params.Json("fields")
	for name := range fields {
		def := fields.Json(name)
		for key := range def {
			if !slices.Contains(conditionKeys, key) {
				return nil, fmt.Errorf(MSG_VALIDATOR_UNKNOWN_CONDITION, name, key)
			}
		}

		pattern := def.Str("pattern")
		if _, err := regexp.Compile(pattern); err != nil {
			return nil, fmt.Errorf(MSG_VALIDATOR_INVALID_PATTERN, name, pattern)
		}

		condition := s.Field(name)
		condition.required = def.Bool("required")
		condition.notEmpty = def.Bool("not_empty")
		condition.isLetters = def.Bool("is_letters")
		condition.isNumbers = def.Bool("is_numbers")
		condition.isSpecialCharacters = def.Bool("is_special_characters")
		condition.isEmail = def.Bool("is_email")
		condition.isPhone = def.Bool("is_phone")
		condition.min = def.Num("min")
		condition.max = def.Num("max")
		condition.minLength = def.Int("min_length")
		condition.maxLength = def.Int("max_length")
		condition.pattern = pattern
	}

	return s, nil
}

/**
* ToJson: The validator as the json ByJson reads (only the conditions that are set)
* @return et.Json
**/
func (s *Validator) ToJson() et.Json {
	fields := et.Json{}
	for name, field := range s.Fields {
		fields[name] = field.Condition.ToJson()
	}

	result := et.Json{"fields": fields}
	if s.notEmpty {
		result["not_empty"] = true
	}

	return result
}

/**
* ToJson: The condition as json (only what is set)
* @return et.Json
**/
func (s *Condition) ToJson() et.Json {
	result := et.Json{}
	flags := map[string]bool{
		"required":              s.required,
		"not_empty":             s.notEmpty,
		"is_letters":            s.isLetters,
		"is_numbers":            s.isNumbers,
		"is_special_characters": s.isSpecialCharacters,
		"is_email":              s.isEmail,
		"is_phone":              s.isPhone,
	}
	for key, on := range flags {
		if on {
			result[key] = true
		}
	}

	if s.min != 0 {
		result["min"] = s.min
	}
	if s.max != 0 {
		result["max"] = s.max
	}
	if s.minLength != 0 {
		result["min_length"] = s.minLength
	}
	if s.maxLength != 0 {
		result["max_length"] = s.maxLength
	}
	if s.pattern != "" {
		result["pattern"] = s.pattern
	}

	return result
}
