package validator

import (
	"regexp"
	"time"

	"github.com/cgalvisleon/et/et"
)

type Validator struct {
	required  bool
	min       float64
	max       float64
	minLength int
	maxLength int
	pattern   string
	message   string
}

func New() *Validator {
	return &Validator{
		required:  false,
		min:       0,
		max:       0,
		minLength: 0,
		maxLength: 0,
		pattern:   "",
		message:   "",
	}
}

/**
* Required: Set the required flag for the validator.
* @param required bool
* @return *Validator
**/
func (v *Validator) Required(required bool) *Validator {
	v.required = required
	return v
}

/**
* Min: Set the minimum value for the validator.
* @param min int
* @return *Validator
**/
func (v *Validator) Min(min float64) *Validator {
	v.min = min
	return v
}

/**
* Max: Set the maximum value for the validator.
* @param max int
* @return *Validator
**/
func (v *Validator) Max(max float64) *Validator {
	v.max = max
	return v
}

/**
* Between: Set the minimum and maximum values for the validator.
* @param min, max int
* @return *Validator
**/
func (v *Validator) Between(min, max float64) *Validator {
	v.min = min
	v.max = max
	return v
}

/**
* MinLength: Set the minimum length for the validator.
* @param minLength int
* @return *Validator
**/
func (v *Validator) MinLength(minLength int) *Validator {
	v.minLength = minLength
	return v
}

/**
* MaxLength: Set the maximum length for the validator.
* @param maxLength int
* @return *Validator
**/
func (v *Validator) MaxLength(maxLength int) *Validator {
	v.maxLength = maxLength
	return v
}

/**
* Pattern: Set the pattern for the validator.
* @param pattern string
* @return *Validator
**/
func (v *Validator) Pattern(pattern string) *Validator {
	v.pattern = pattern
	return v
}

/**
* Message: Set the message for the validator.
* @param message string
* @return *Validator
**/
func (v *Validator) Message(message string) *Validator {
	v.message = message
	return v
}

/**
* Validate: Validate the value using the validator.
* @param value any
* @return bool
**/
func (v *Validator) Validate(value any) bool {
	switch value.(type) {
	case string:
		return v.validateString(value.(string))
	case int:
		return v.validateInt(value.(int))
	case float64:
		return v.validateFloat(value.(float64))
	case bool:
		return v.validateBool(value.(bool))
	case time.Time:
		return v.validateTime(value.(time.Time))
	case []byte:
		return v.validateBytes(value.([]byte))
	case []string:
		return v.validateArrayString(value.([]string))
	case []int:
		return v.validateInts(value.([]int))
	case []float64:
		return v.validateFloats(value.([]float64))
	case map[string]interface{}:
		return v.validateJson(value.(map[string]interface{}))
	case et.Json:
		return v.validateJson(value.(et.Json))
	case []et.Json:
		return v.validateArrayJson(value.([]et.Json))
	}
	return false
}

/**
* validateString: Validate the string value using the validator.
* @param value string
* @return bool
**/
func (v *Validator) validateString(value string) bool {
	if v.required && value == "" {
		return false
	} else if v.minLength > 0 && len(value) < v.minLength {
		return false
	} else if v.maxLength > 0 && len(value) > v.maxLength {
		return false
	} else if v.pattern != "" && !regexp.MustCompile(v.pattern).MatchString(value) {
		return false
	}
	return true
}

/**
* validateInt: Validate the int value using the validator.
* @param value int
* @return bool
**/
func (v *Validator) validateInt(value int) bool {
	if v.required && value == 0 {
		return false
	} else if v.min > 0 && float64(value) < v.min {
		return false
	} else if v.max > 0 && float64(value) > v.max {
		return false
	}
	return true
}

/**
* validateFloat: Validate the float value using the validator.
* @param value float64
* @return bool
**/
func (v *Validator) validateFloat(value float64) bool {
	if v.required && value == 0 {
		return false
	} else if v.min > 0 && value < v.min {
		return false
	} else if v.max > 0 && value > v.max {
		return false
	}
	return true
}

/**
* validateBool: Validate the bool value using the validator.
* @param value bool
* @return bool
**/
func (v *Validator) validateBool(value bool) bool {
	if v.required && !value {
		return false
	}
	return true
}

/**
* validateTime: Validate the time value using the validator.
* @param value time.Time
* @return bool
**/
func (v *Validator) validateTime(value time.Time) bool {
	if v.required && value.IsZero() {
		return false
	}
	return true
}

/**
* validateBytes: Validate the bytes value using the validator.
* @param value []byte
* @return bool
**/
func (v *Validator) validateBytes(value []byte) bool {
	if v.required && len(value) == 0 {
		return false
	} else if v.minLength > 0 && len(value) < v.minLength {
		return false
	} else if v.maxLength > 0 && len(value) > v.maxLength {
		return false
	}
	return true
}

/**
* validateArrayString: Validate the array of strings value using the validator.
* @param value []string
* @return bool
**/
func (v *Validator) validateArrayString(value []string) bool {
	if v.required && len(value) == 0 {
		return false
	} else if v.minLength > 0 && len(value) < v.minLength {
		return false
	} else if v.maxLength > 0 && len(value) > v.maxLength {
		return false
	}
	return true
}

/**
* validateInts: Validate the array of ints value using the validator.
* @param value []int
* @return bool
**/
func (v *Validator) validateInts(value []int) bool {
	if v.required && len(value) == 0 {
		return false
	} else if v.minLength > 0 && len(value) < v.minLength {
		return false
	} else if v.maxLength > 0 && len(value) > v.maxLength {
		return false
	}
	return true
}

/**
* validateFloats: Validate the array of floats value using the validator.
* @param value []float64
* @return bool
**/
func (v *Validator) validateFloats(value []float64) bool {
	if v.required && len(value) == 0 {
		return false
	} else if v.minLength > 0 && len(value) < v.minLength {
		return false
	} else if v.maxLength > 0 && len(value) > v.maxLength {
		return false
	}
	return true
}

/**
* validateArrayInt: Validate the array of ints value using the validator.
* @param value []int
* @return bool
**/
func (v *Validator) validateArrayInt(value []int) bool {
	if v.required && len(value) == 0 {
		return false
	} else if v.minLength > 0 && len(value) < v.minLength {
		return false
	} else if v.maxLength > 0 && len(value) > v.maxLength {
		return false
	}
	return true
}

/**
* validateArrayFloat: Validate the array of floats value using the validator.
* @param value []float64
* @return bool
**/
func (v *Validator) validateArrayFloat(value []float64) bool {
	if v.required && len(value) == 0 {
		return false
	} else if v.minLength > 0 && len(value) < v.minLength {
		return false
	} else if v.maxLength > 0 && len(value) > v.maxLength {
		return false
	}
	return true
}

/**
* validateJson: Validate the json value using the validator.
* @param value map[string]interface{}
* @return bool
**/
func (v *Validator) validateJson(value map[string]interface{}) bool {
	if v.required && len(value) == 0 {
		return false
	}
	return true
}

/**
* validateArrayJson: Validate the array of json value using the validator.
* @param value []et.Json
* @return bool
**/
func (v *Validator) validateArrayJson(value []et.Json) bool {
	if v.required && len(value) == 0 {
		return false
	}
	return true
}
