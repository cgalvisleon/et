package validator

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
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
	min                 float64
	max                 float64
	minLength           int
	maxLength           int
	pattern             string
	name                string
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
* AddField: Add a field to the validator.
* @param name string, validator *Condition
* @return *Validator
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
* Fields: Get the fields of the validator.
* @return map[string]*Field
**/
func (s *Condition) Field(name string) *Condition {
	return s.validator.Field(name)
}

/**
* Required: Set the required flag for the validator.
* @param required bool
* @return *Validator
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
* Min: Set the minimum value for the validator.
* @param min int
* @return *Validator
**/
func (s *Condition) Min(min float64) *Condition {
	s.min = min
	return s
}

/**
* Max: Set the maximum value for the validator.
* @param max int
* @return *Validator
**/
func (s *Condition) Max(max float64) *Condition {
	s.max = max
	return s
}

/**
* Between: Set the minimum and maximum values for the validator.
* @param min, max int
* @return *Validator
**/
func (s *Condition) Between(min, max float64) *Condition {
	s.min = min
	s.max = max
	return s
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
* Pattern: Set the pattern for the validator.
* @param pattern string
* @return *Validator
**/
func (s *Condition) Pattern(pattern string) *Condition {
	s.pattern = pattern
	return s
}

/**
* Validate: Validate the value using the validator.
* @param value any
* @return bool
**/
func (s *Condition) validate(value any) (bool, error) {
	switch v := value.(type) {
	case string:
		return s.validateString(v)
	case int:
		return s.validateInt(v)
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
	} else if s.pattern != "" && !regexp.MustCompile(s.pattern).MatchString(value) {
		return false, fmt.Errorf(MSG_VALIDATOR_PATTERN, s.name, s.pattern)
	} else if s.isLetters && !letterPattern.MatchString(value) {
		return false, fmt.Errorf(MSG_VALIDATOR_LETTERS, s.name)
	} else if s.isNumbers && !numberPattern.MatchString(value) {
		return false, fmt.Errorf(MSG_VALIDATOR_NUMBERS, s.name)
	} else if s.isSpecialCharacters && !specialCharPattern.MatchString(value) {
		return false, fmt.Errorf(MSG_VALIDATOR_SPECIAL_CHARACTERS, s.name)
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
	} else if s.min > 0 && float64(value) < s.min {
		return false, fmt.Errorf(MSG_VALIDATOR_MIN, s.name, s.min)
	} else if s.max > 0 && float64(value) > s.max {
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
	} else if s.min > 0 && value < s.min {
		return false, fmt.Errorf(MSG_VALIDATOR_MIN, s.name, s.min)
	} else if s.max > 0 && value > s.max {
		return false, fmt.Errorf(MSG_VALIDATOR_MAX, s.name, s.max)
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
