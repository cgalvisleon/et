package utility

import (
	"regexp"

	"github.com/cgalvisleon/et/et"
	"golang.org/x/exp/slices"
)

func ValidStr(val string, min int, notIn []string) bool {
	v := et.Replace(val, " ", "")
	ok := len(v) > min
	if ok {
		return ok
	}

	ok = Contains(notIn, val)
	return !ok
}

func ValidIn(val string, min int, in []string) bool {
	v := et.Replace(val, " ", "")
	ok := len(v) > min && Contains(in, val)
	return ok
}

func ValidId(val string) bool {
	ok := Contains([]string{"", "*", "new"}, val)
	return !ok
}

func ValidInt(val int, notIn []int) bool {
	ok := slices.Contains(notIn, val)

	return !ok
}

func ValidNum(val float64, notIn []float64) bool {
	ok := slices.Contains(notIn, val)

	return !ok
}

func ValidName(val string) bool {
	regex := `^[a-zA-Z\s\']+`
	pattern := regexp.MustCompile(regex)
	return pattern.MatchString(val)
}

func ValidEmail(val string) bool {
	regex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	pattern := regexp.MustCompile(regex)
	return pattern.MatchString(val)
}

func ValidPhone(val string) bool {
	regex := `^\d{10}$`
	pattern := regexp.MustCompile(regex)
	return pattern.MatchString(val)
}

func ValidUUID(val string) bool {
	regex := `^(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	pattern := regexp.MustCompile(regex)
	return pattern.MatchString(val)
}

func ValidCode(val string) bool {
	regex := `^-?\d+$`
	pattern := regexp.MustCompile(regex)
	ok := len(val) >= 6 && pattern.MatchString(val)
	return ok
}
