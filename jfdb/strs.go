package jfdb

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

func Format(format string, args ...any) string {
	result := fmt.Sprintf(format, args...)

	return result
}

func FormatUppCase(format string, args ...any) string {
	result := Format(format, args...)

	return Uppcase(result)
}

func FormatLowCase(format string, args ...any) string {
	result := Format(format, args...)

	return Lowcase(result)
}

func Replace(str string, old string, new string) string {
	return strings.ReplaceAll(str, old, new)
}

func ReplaceAll(str string, olds []string, new string) string {
	var result string = str
	for _, str := range olds {
		result = strings.ReplaceAll(result, str, new)
	}

	return result
}

func Name(str string) string {
	regex := `[0-9\s]+`
	pattern := regexp.MustCompile(regex)
	return pattern.ReplaceAllString(str, "_")
}

func Trim(str string) string {
	return strings.Trim(str, " ")
}

func NotSpace(str string) string {
	return Replace(str, " ", "")
}

func Uppcase(s string) string {
	return strings.ToUpper(s)
}

func Lowcase(s string) string {
	return strings.ToLower(s)
}

func Titlecase(str string) string {
	var result string
	var ok bool
	for i, char := range str {
		s := fmt.Sprintf("%c", char)
		if i == 0 {
			s = strings.ToUpper(s)
		} else if s == "" {
			ok = true
		} else if ok {
			ok = false
			s = strings.ToUpper(s)
		}

		result = Append(result, s, "")
	}

	return result
}

func Empty(str1, str2 string) string {
	if len(str1) == 0 {
		return str2
	}

	return str1
}

func Append(str1, str2, sp string) string {
	if len(str1) == 0 {
		return str2
	}
	if len(str2) == 0 {
		return str1
	}

	return Format(`%s%s%s`, str1, sp, str2)
}

func AppendStr(s1, s2 string) string {
	if len(s2) == 0 {
		return s1
	}
	if len(s1) == 0 {
		return s2
	}

	return Format(`%s_%s`, strings.ToUpper(s1), strings.ToUpper(s2))
}

func Split(str, sep string) []string {
	return strings.Split(str, sep)
}

func GetSplitIndex(str, sep string, idx int) string {
	split := Split(str, sep)
	if idx < 0 {
		idx = len(split) + idx
	}

	if idx < len(split) {
		return split[idx]
	}

	return ""
}

func ApendAny(space string, args ...any) string {
	var result string = ""
	for i, a := range args {
		if i == 0 {
			result = fmt.Sprintf(`%v`, a)
		} else if len(result) == 0 && len(fmt.Sprint(a)) > 0 {
			result = fmt.Sprintf(`%v`, a)
		} else if len(result) > 0 && len(fmt.Sprint(a)) > 0 {
			result = fmt.Sprintf(`%s%v%v`, result, space, a)
		}
	}

	return result
}

func StrToTime(val string) (time.Time, error) {
	var result time.Time
	layout := "2006-01-02T15:04:05.000Z"

	result, err := time.Parse(layout, val)
	if err != nil {
		return result, err
	}

	return result, nil
}

func StrToBool(val string) (bool, error) {
	if Lowcase(val) == "true" {
		return true, nil
	} else if Lowcase(val) == "false" {
		return false, nil
	}

	return false, errors.New("invalid boolean value")
}

func RemoveAcents(str string) string {
	str = strings.ReplaceAll(str, "á", "a")
	str = strings.ReplaceAll(str, "é", "e")
	str = strings.ReplaceAll(str, "í", "i")
	str = strings.ReplaceAll(str, "ó", "o")
	str = strings.ReplaceAll(str, "ú", "u")

	str = strings.ReplaceAll(str, "Á", "A")
	str = strings.ReplaceAll(str, "É", "E")
	str = strings.ReplaceAll(str, "Í", "I")
	str = strings.ReplaceAll(str, "Ó", "O")
	str = strings.ReplaceAll(str, "Ú", "U")
	return str
}
