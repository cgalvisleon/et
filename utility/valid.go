package utility

import (
	"fmt"
	"regexp"

	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
)

/**
* validate
* @param expr, value string
* @return bool
**/
func validate(expr, val string) bool {
	re := regexp.MustCompile(expr)
	return re.MatchString(val)
}

/**
* Validate
* @param keys []string
* @return error
**/
func Validate(keys []string) error {
	for _, key := range keys {
		if key == "" {
			return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, key)
		}
	}
	return nil
}

/**
* ValidStr
* @param val string, min int, notIn []string
* @return bool
**/
func ValidStr(val string, min int, notIn []string) bool {
	v := strs.Replace(val, " ", "")
	ok := len(v) > min && !Contains(val, notIn)
	return ok
}

/**
* ValidIn
* @param val string, min int, in []string
* @return bool
**/
func ValidIn(val string, min int, in []string) bool {
	v := strs.Replace(val, " ", "")
	ok := len(v) > min && Contains(val, in)
	return ok
}

/**
* ValidId
* @param val string
* @return bool
**/
func ValidId(val string) bool {
	return ValidStr(val, 0, []string{"", "*", "new"})
}

/**
* ValidKey
* @param val string
* @return bool
**/
func ValidKey(val string) bool {
	return ValidStr(val, 0, []string{"-1", "", "*", "new"})
}

/**
* ValidInt
* @param val int, notIn []int
* @return bool
**/
func ValidInt(val int, notIn []int) bool {
	ok := Contains(val, notIn)

	return !ok
}

/**
* ValidNum
* @param val float64, notIn []float64
* @return bool
**/
func ValidNum(val float64, notIn []float64) bool {
	ok := Contains(val, notIn)

	return !ok
}

/**
* ValidName
* @param val string
* @return bool
**/
func ValidName(val string) bool {
	return validate(`^[a-zA-Z\s\']+`, val)
}

/**
* ValidEmail
* @param val string
* @return bool
**/
func ValidEmail(val string) bool {
	return validate(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, val)
}

/**
* ValidPhone
* @param val string
* @return bool
**/
func ValidPhone(val string) bool {
	return validate(`^\d{10}$`, val)
}

/**
* ValidUUID
* @param val string
* @return bool
**/
func ValidUUID(val string) bool {
	return validate(`^(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, val)
}

/**
* ValidCode
* @param val string
* @return bool
**/

func ValidCode(val string) bool {
	return validate(`^\d{6,}$`, val)
}

/**
* ValidWord
* @param word string
* @return bool
**/
func ValidWord(word string) bool {
	return validate(`^[a-zA-ZáéíóúÁÉÍÓÚñÑ0-9]+$`, word)
}
