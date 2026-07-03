package validator

import (
	"regexp"
	"time"

	"github.com/cgalvisleon/et/et"
)

type Condition struct {
	required  bool
	min       float64
	max       float64
	minLength int
	maxLength int
	pattern   string
	message   string
}

type Field struct {
	Name      string
	Validator *Condition
}

type Validator struct {
	Fields map[string]*Field
}

func New() *Validator {
	return &Validator{
		Fields: make(map[string]*Field),
	}
}

/**
* Required: Set the required flag for the validator.
* @param required bool
* @return *Validator
**/
func (s *Condition) Required(required bool) *Condition {
	s.required = required
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
* Message: Set the message for the validator.
* @param message string
* @return *Validator
**/
func (s *Condition) Message(message string) *Condition {
	s.message = message
	return s
}

/**
* Validate: Validate the value using the validator.
* @param value any
* @return bool
**/
func (s *Condition) validate(value any) bool {
	switch value.(type) {
	case string:
		return s.validateString(value.(string))
	case int:
		return s.validateInt(value.(int))
	case float64:
		return s.validateFloat(value.(float64))
	case bool:
		return s.validateBool(value.(bool))
	case time.Time:
		return s.validateTime(value.(time.Time))
	case []byte:
		return s.validateBytes(value.([]byte))
	case []string:
		return s.validateArrayString(value.([]string))
	case []int:
		return s.validateInts(value.([]int))
	case []float64:
		return s.validateFloats(value.([]float64))
	}
	return false
}

/**
* validateString: Validate the string value using the validator.
* @param value string
* @return bool
**/
func (s *Condition) validateString(value string) bool {
	if s.required && value == "" {
		return false
	} else if s.minLength > 0 && len(value) < s.minLength {
		return false
	} else if s.maxLength > 0 && len(value) > s.maxLength {
		return false
	} else if s.pattern != "" && !regexp.MustCompile(s.pattern).MatchString(value) {
		return false
	}
	return true
}

/**
* validateInt: Validate the int value using the validator.
* @param value int
* @return bool
**/
func (s *Condition) validateInt(value int) bool {
	if s.required && value == 0 {
		return false
	} else if s.min > 0 && float64(value) < s.min {
		return false
	} else if s.max > 0 && float64(value) > s.max {
		return false
	}
	return true
}

/**
* validateFloat: Validate the float value using the validator.
* @param value float64
* @return bool
**/
func (s *Condition) validateFloat(value float64) bool {
	if s.required && value == 0 {
		return false
	} else if s.min > 0 && value < s.min {
		return false
	} else if s.max > 0 && value > s.max {
		return false
	}
	return true
}

/**
* validateBool: Validate the bool value using the validator.
* @param value bool
* @return bool
**/
func (s *Condition) validateBool(value bool) bool {
	if s.required && !value {
		return false
	}
	return true
}

/**
* validateTime: Validate the time value using the validator.
* @param value time.Time
* @return bool
**/
func (s *Condition) validateTime(value time.Time) bool {
	if s.required && value.IsZero() {
		return false
	}
	return true
}

/**
* validateBytes: Validate the bytes value using the validator.
* @param value []byte
* @return bool
**/
func (s *Condition) validateBytes(value []byte) bool {
	if s.required && len(value) == 0 {
		return false
	} else if s.minLength > 0 && len(value) < s.minLength {
		return false
	} else if s.maxLength > 0 && len(value) > s.maxLength {
		return false
	}
	return true
}

/**
* validateArrayString: Validate the array of strings value using the validator.
* @param value []string
* @return bool
**/
func (s *Condition) validateArrayString(value []string) bool {
	if s.required && len(value) == 0 {
		return false
	} else if s.minLength > 0 && len(value) < s.minLength {
		return false
	} else if s.maxLength > 0 && len(value) > s.maxLength {
		return false
	}
	return true
}

/**
* validateInts: Validate the array of ints value using the validator.
* @param value []int
* @return bool
**/
func (s *Condition) validateInts(value []int) bool {
	if s.required && len(value) == 0 {
		return false
	} else if s.minLength > 0 && len(value) < s.minLength {
		return false
	} else if s.maxLength > 0 && len(value) > s.maxLength {
		return false
	}
	return true
}

/**
* validateFloats: Validate the array of floats value using the validator.
* @param value []float64
* @return bool
**/
func (s *Condition) validateFloats(value []float64) bool {
	if s.required && len(value) == 0 {
		return false
	} else if s.minLength > 0 && len(value) < s.minLength {
		return false
	} else if s.maxLength > 0 && len(value) > s.maxLength {
		return false
	}
	return true
}

/**
* validateArrayFloat: Validate the array of floats value using the validator.
* @param value []float64
* @return bool
**/
func (s *Condition) validateArrayFloat(value []float64) bool {
	if s.required && len(value) == 0 {
		return false
	} else if s.minLength > 0 && len(value) < s.minLength {
		return false
	} else if s.maxLength > 0 && len(value) > s.maxLength {
		return false
	}
	return true
}

/**
* Validate: Validate the json value using the validator.
* @param value et.Json
* @return bool
**/
func (s *Validator) Validate(value et.Json) bool {
	for key, val := range value {
		if s.Fields[key].Validator.validate(val) {
			return false
		}
	}
	return true
}
