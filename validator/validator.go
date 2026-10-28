package validator

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

var letterPattern = regexp.MustCompile(`\p{L}`)
var numberPattern = regexp.MustCompile(`[0-9]`)
var specialCharPattern = regexp.MustCompile(`[^\p{L}0-9\s]`)

type Condition struct {
	required            bool
	notEmpty            bool
	isLetters           bool
	isNumbers           bool
	isSpecialCharacters bool
	isEmail             bool
	isPhone             bool
	min                 float64
	max                 float64
	hasMin              bool
	hasMax              bool
	minLength           int
	maxLength           int
	pattern             string
	re                  *regexp.Regexp
	patternErr          error
	name                string
	values              []any
	validator           *Validator
}

type Field struct {
	Name      string     `json:"name"`
	Condition *Condition `json:"condition"`
}

type Validator struct {
	Fields   map[string]*Field
	notEmpty bool
}

/**
* New: An empty validator; add fields with Field.
* @return *Validator
**/
func New() *Validator {
	return &Validator{
		Fields: make(map[string]*Field),
	}
}

/**
* NotEmpty: Require the json data passed to Validate to contain at least one key.
* @return *Validator
**/
func (s *Validator) NotEmpty() *Validator {
	s.notEmpty = true
	return s
}

/**
* Field: Adds a field to the validator and returns its condition to chain the checks.
* @param name string
* @return *Condition
**/
func (s *Validator) Field(name string) *Condition {
	s.Fields[name] = &Field{
		Name: name,
		Condition: &Condition{
			name:      name,
			validator: s,
		},
	}
	return s.Fields[name].Condition
}

/**
* Field: Adds another field to the same validator, to keep chaining.
* @param name string
* @return *Condition
**/
func (s *Condition) Field(name string) *Condition {
	return s.validator.Field(name)
}

/**
* Required: Require the field to be present and not empty.
* @return *Condition
**/
func (s *Condition) Required() *Condition {
	s.required = true
	return s
}

/**
* NotEmpty: Require the string value to contain at least one non-whitespace character.
* @return *Condition
**/
func (s *Condition) NotEmpty() *Condition {
	s.notEmpty = true
	return s
}

/**
* IsLetters: Require the string value to contain at least one letter. Can be combined with IsNumbers and IsSpecialCharacters.
* @return *Condition
**/
func (s *Condition) IsLetters() *Condition {
	s.isLetters = true
	return s
}

/**
* IsNumbers: Require the string value to contain at least one digit. Can be combined with IsLetters and IsSpecialCharacters.
* @return *Condition
**/
func (s *Condition) IsNumbers() *Condition {
	s.isNumbers = true
	return s
}

/**
* IsSpecialCharacters: Require the string value to contain at least one special (non-alphanumeric) character. Can be combined with IsLetters and IsNumbers.
* @return *Condition
**/
func (s *Condition) IsSpecialCharacters() *Condition {
	s.isSpecialCharacters = true
	return s
}

/**
* IsEmail: Require the string value to be a valid email address.
* @return *Condition
**/
func (s *Condition) IsEmail() *Condition {
	s.isEmail = true
	return s
}

/**
* IsPhone: Require the string value to be a valid phone number.
* @return *Condition
**/
func (s *Condition) IsPhone() *Condition {
	s.isPhone = true
	return s
}

/**
* Min: Require a numeric value to be at least min.
* @param min float64
* @return *Condition
**/
func (s *Condition) Min(min float64) *Condition {
	s.min = min
	s.hasMin = true
	return s
}

/**
* Max: Require a numeric value to be at most max.
* @param max float64
* @return *Condition
**/
func (s *Condition) Max(max float64) *Condition {
	s.max = max
	s.hasMax = true
	return s
}

/**
* Between: Set the minimum and maximum values for the validator.
* @param min, max int
* @return *Validator
**/
func (s *Condition) Between(min, max float64) *Condition {
	return s.Min(min).Max(max)
}

/**
* MinLength: Set the minimum length for the validator.
* @param minLength int
* @return *Validator
**/
func (s *Condition) MinLength(minLength int) *Condition {
	s.minLength = minLength
	return s
}

/**
* MaxLength: Set the maximum length for the validator.
* @param maxLength int
* @return *Validator
**/
func (s *Condition) MaxLength(maxLength int) *Condition {
	s.maxLength = maxLength
	return s
}

/**
* Pattern: Set the regular expression the value must match; compiled once here, an invalid one fails every validation.
* @param pattern string
* @return *Condition
**/
func (s *Condition) Pattern(pattern string) *Condition {
	s.pattern = pattern
	s.re, s.patternErr = regexp.Compile(pattern)
	if s.patternErr != nil {
		s.patternErr = fmt.Errorf(MSG_VALIDATOR_INVALID_PATTERN, s.name, pattern)
	}
	return s
}

/**
* Values: Restrict the value to one of these values; on a list, each of its items.
* @param values ...any
* @return *Condition
**/
func (s *Condition) Values(values ...any) *Condition {
	s.values = values
	return s
}

/**
* validate: Validate the value by its type and then against the allowed values.
* @param value any
* @return bool, error
**/
func (s *Condition) validate(value any) (bool, error) {
	ok, err := s.validateType(value)
	if !ok || err != nil {
		return ok, err
	}
	return s.validateValues(value)
}

/**
* validateValues: Validate that the value (or each item of a list) is one of the allowed values; nil passes.
* @param value any
* @return bool, error
**/
func (s *Condition) validateValues(value any) (bool, error) {
	if len(s.values) == 0 || value == nil {
		return true, nil
	}

	items := []any{value}
	switch v := value.(type) {
	case []string:
		items = toAny(v)
	case []int:
		items = toAny(v)
	case []float64:
		items = toAny(v)
	case []any:
		items = v
	case et.Json, map[string]any, []et.Json, []byte:
		return true, nil
	}

	for _, item := range items {
		if !slices.ContainsFunc(s.values, func(allowed any) bool { return sameValue(item, allowed) }) {
			return false, fmt.Errorf(MSG_VALIDATOR_VALUES, s.name, s.values)
		}
	}
	return true, nil
}

/**
* toAny: A typed list as []any.
* @param values []T
* @return []any
**/
func toAny[T any](values []T) []any {
	result := make([]any, len(values))
	for i, value := range values {
		result[i] = value
	}
	return result
}

/**
* sameValue: Whether two values are equal; numbers compare by value whatever their type (JSON gives float64, goja int64).
* @param a, b any
* @return bool
**/
func sameValue(a, b any) bool {
	numA, okA := toFloat(a)
	numB, okB := toFloat(b)
	if okA || okB {
		return okA && okB && numA == numB
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

/**
* toFloat: The number as float64, and whether it is a number.
* @param value any
* @return float64, bool
**/
func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	}
	return 0, false
}

/**
* validateType: Validate the value by its type.
* @param value any
* @return bool, error
**/
func (s *Condition) validateType(value any) (bool, error) {
	switch v := value.(type) {
	case nil:
		return s.validateNil()
	case string:
		return s.validateString(v)
	case int:
		return s.validateInt(v)
	case int32:
		return s.validateInt(int(v))
	case int64:
		// goja exporta los enteros de JavaScript como int64
		return s.validateInt(int(v))
	case float32:
		return s.validateFloat(float64(v))
	case float64:
		return s.validateFloat(v)
	case bool:
		return s.validateBool(v)
	case time.Time:
		return s.validateTime(v)
	case []byte:
		return s.validateBytes(v)
	case []string:
		return s.validateArrayString(v)
	case []int:
		return s.validateInts(v)
	case []float64:
		return s.validateFloats(v)
	case []any:
		return s.validateArrayAny(v)
	case et.Json:
		return s.validateArrayLength(len(v))
	case map[string]any:
		return s.validateArrayLength(len(v))
	case []et.Json:
		return s.validateArrayLength(len(v))
	}
	return false, fmt.Errorf(MSG_VALIDATOR_INVALID_TYPE, s.name)
}

/**
* validateString: Validate the string value using the validator.
* @param value string
* @return bool, error
**/
func (s *Condition) validateString(value string) (bool, error) {
	if s.required && value == "" {
		return false, fmt.Errorf(MSG_VALIDATOR_REQUIRED, s.name)
	} else if s.notEmpty && strings.TrimSpace(value) == "" {
		return false, fmt.Errorf(MSG_VALIDATOR_NOT_EMPTY, s.name)
	} else if s.minLength > 0 && len(value) < s.minLength {
		return false, fmt.Errorf(MSG_VALIDATOR_MIN_LENGTH, s.name, s.minLength)
	} else if s.maxLength > 0 && len(value) > s.maxLength {
		return false, fmt.Errorf(MSG_VALIDATOR_MAX_LENGTH, s.name, s.maxLength)
	} else if s.patternErr != nil {
		return false, s.patternErr
	} else if s.re != nil && !s.re.MatchString(value) {
		return false, fmt.Errorf(MSG_VALIDATOR_PATTERN, s.name, s.pattern)
	} else if s.isLetters && !letterPattern.MatchString(value) {
		return false, fmt.Errorf(MSG_VALIDATOR_LETTERS, s.name)
	} else if s.isNumbers && !numberPattern.MatchString(value) {
		return false, fmt.Errorf(MSG_VALIDATOR_NUMBERS, s.name)
	} else if s.isSpecialCharacters && !specialCharPattern.MatchString(value) {
		return false, fmt.Errorf(MSG_VALIDATOR_SPECIAL_CHARACTERS, s.name)
	} else if s.isEmail && !utility.ValidEmail(value) {
		return false, fmt.Errorf(MSG_VALIDATOR_EMAIL, s.name)
	} else if s.isPhone && !utility.ValidPhone(value) {
		return false, fmt.Errorf(MSG_VALIDATOR_PHONE, s.name)
	}
	return true, nil
}

/**
* validateInt: Validate the int value using the validator.
* @param value int
* @return bool, error
**/
func (s *Condition) validateInt(value int) (bool, error) {
	if s.required && value == 0 {
		return false, fmt.Errorf(MSG_VALIDATOR_REQUIRED, s.name)
	} else if s.hasMin && float64(value) < s.min {
		return false, fmt.Errorf(MSG_VALIDATOR_MIN, s.name, s.min)
	} else if s.hasMax && float64(value) > s.max {
		return false, fmt.Errorf(MSG_VALIDATOR_MAX, s.name, s.max)
	}
	return true, nil
}

/**
* validateFloat: Validate the float value using the validator.
* @param value float64
* @return bool, error
**/
func (s *Condition) validateFloat(value float64) (bool, error) {
	if s.required && value == 0 {
		return false, fmt.Errorf(MSG_VALIDATOR_REQUIRED, s.name)
	} else if s.hasMin && value < s.min {
		return false, fmt.Errorf(MSG_VALIDATOR_MIN, s.name, s.min)
	} else if s.hasMax && value > s.max {
		return false, fmt.Errorf(MSG_VALIDATOR_MAX, s.name, s.max)
	}
	return true, nil
}

/**
* validateNil: Validate a missing or null value: only fails if the field is required.
* @return bool, error
**/
func (s *Condition) validateNil() (bool, error) {
	if s.required {
		return false, fmt.Errorf(MSG_VALIDATOR_REQUIRED, s.name)
	}
	return true, nil
}

/**
* validateBool: Validate the bool value using the validator.
* @param value bool
* @return bool, error
**/
func (s *Condition) validateBool(value bool) (bool, error) {
	if s.required && !value {
		return false, fmt.Errorf(MSG_VALIDATOR_REQUIRED, s.name)
	}
	return true, nil
}

/**
* validateTime: Validate the time value using the validator.
* @param value time.Time
* @return bool, error
**/
func (s *Condition) validateTime(value time.Time) (bool, error) {
	if s.required && value.IsZero() {
		return false, fmt.Errorf(MSG_VALIDATOR_REQUIRED, s.name)
	}
	return true, nil
}

/**
* validateBytes: Validate the bytes value using the validator.
* @param value []byte
* @return bool, error
**/
func (s *Condition) validateBytes(value []byte) (bool, error) {
	if s.required && len(value) == 0 {
		return false, fmt.Errorf(MSG_VALIDATOR_REQUIRED, s.name)
	} else if s.minLength > 0 && len(value) < s.minLength {
		return false, fmt.Errorf(MSG_VALIDATOR_MIN_LENGTH, s.name, s.minLength)
	} else if s.maxLength > 0 && len(value) > s.maxLength {
		return false, fmt.Errorf(MSG_VALIDATOR_MAX_LENGTH, s.name, s.maxLength)
	}
	return true, nil
}

/**
* validateArrayLength: Validate an array length against required/minLength/maxLength.
* @param length int
* @return bool, error
**/
func (s *Condition) validateArrayLength(length int) (bool, error) {
	if s.required && length == 0 {
		return false, fmt.Errorf(MSG_VALIDATOR_REQUIRED, s.name)
	} else if s.minLength > 0 && length < s.minLength {
		return false, fmt.Errorf(MSG_VALIDATOR_MIN_LENGTH, s.name, s.minLength)
	} else if s.maxLength > 0 && length > s.maxLength {
		return false, fmt.Errorf(MSG_VALIDATOR_MAX_LENGTH, s.name, s.maxLength)
	}
	return true, nil
}

/**
* validateArrayString: Validate the array of strings value using the validator.
* @param value []string
* @return bool, error
**/
func (s *Condition) validateArrayString(value []string) (bool, error) {
	return s.validateArrayLength(len(value))
}

/**
* validateInts: Validate the array of ints value using the validator.
* @param value []int
* @return bool, error
**/
func (s *Condition) validateInts(value []int) (bool, error) {
	return s.validateArrayLength(len(value))
}

/**
* validateFloats: Validate the array of floats value using the validator.
* @param value []float64
* @return bool, error
**/
func (s *Condition) validateFloats(value []float64) (bool, error) {
	return s.validateArrayLength(len(value))
}

/**
* validateArrayAny: Validate a generic array value (e.g. []any from a decoded JSON array) using the validator.
* @param value []any
* @return bool, error
**/
func (s *Condition) validateArrayAny(value []any) (bool, error) {
	return s.validateArrayLength(len(value))
}

/**
* Validate: Validate the value using the validator.
* @param value any
* @return bool, error
**/
func (s *Condition) Validate(value et.Json) (bool, error) {
	return s.validator.Validate(value)
}

/**
* Validate: Validate the json value using the validator.
* @param value et.Json
* @return bool
**/
func (s *Validator) Validate(value et.Json) (bool, error) {
	if s.notEmpty && value.IsEmpty() {
		return false, errors.New(MSG_VALIDATOR_EMPTY)
	}
	// A required field that does not come in the json fails like an empty one
	for key, field := range s.Fields {
		if _, exists := value[key]; !exists && field.Condition.required {
			return false, fmt.Errorf(MSG_VALIDATOR_REQUIRED, key)
		}
	}
	for key, val := range value {
		field, exists := s.Fields[key]
		if !exists {
			continue
		}

		ok, err := field.Condition.validate(val)
		if !ok || err != nil {
			return false, err
		}
	}
	return true, nil
}
