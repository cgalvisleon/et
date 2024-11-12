package strs

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

/**
* Format
* @param format string
* @param args ...any
* @return string
**/
func Format(format string, args ...any) string {
	result := fmt.Sprintf(format, args...)

	return result
}

/**
* FormatUppCase
* @param format string
* @param args ...any
* @return string
**/
func FormatUppCase(format string, args ...any) string {
	result := Format(format, args...)

	return Uppcase(result)
}

/**
* FormatLowCase
* @param format string
* @param args ...any
* @return string
**/
func FormatLowCase(format string, args ...any) string {
	result := Format(format, args...)

	return Lowcase(result)
}

/**
* FormatDateTime
* @param format string
* @param value time.Time
* @return string
*
* Format examples:
* "2006-01-02" → YYYY-MM-DD
* "02/01/2006" → DD/MM/YYYY
* "15:04:05" → HH:MM:SS (24 horas)
* "03:04:05 PM" → HH:MM:SS AM/PM
*
**/
func FormatDateTime(format string, value time.Time) string {
	if format == "" {
		format = "2006-01-02 15:04:05.000Z"
	}

	return value.Format(format)
}

/**
* Contains
* @param str string
* @param substr string
* @return bool
**/
func Contains(str string, substr string) bool {
	return strings.Contains(str, substr)
}

/**
* Replace
* @param str string
* @param old string
* @param new string
* @return string
**/
func Replace(str string, old string, new string) string {
	return strings.ReplaceAll(str, old, new)
}

/**
* ReplaceAll
* @param str string
* @param olds []string
* @param new string
* @return string
**/
func ReplaceAll(str string, olds []string, new string) string {
	var result string = str
	for _, str := range olds {
		result = strings.ReplaceAll(result, str, new)
	}

	return result
}

/**
* Name
* @param str string
* @return string
**/
func Name(str string) string {
	regex := `[0-9\s]+`
	pattern := regexp.MustCompile(regex)
	return pattern.ReplaceAllString(str, "_")
}

/**
* Trim
* @param str string
* @return string
**/
func Trim(str string) string {
	return strings.Trim(str, " ")
}

/**
* NotSpace
* @param str string
* @return string
**/
func NotSpace(str string) string {
	return Replace(str, " ", "")
}

/**
* DaskSpace
* @param str string
* @return string
**/
func DaskSpace(str string) string {
	return Replace(str, " ", "-")
}

/**
* Uppcase
* @param s string
* @return string
**/
func Uppcase(s string) string {
	return strings.ToUpper(s)
}

/**
* Lowcase
* @param s string
* @return string
**/
func Lowcase(s string) string {
	return strings.ToLower(s)
}

/**
* Titlecase
* @param str string
* @return string
**/
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

/**
* Empty
* @param str1 string
* @param str2 string
* @return string
**/
func Empty(str1, str2 string) string {
	if len(str1) == 0 {
		return str2
	}

	return str1
}

/**
* EmptyAny
* @param val1 interface{}
* @param val2 interface{}
* @return interface{}
**/
func Append(str1, str2, sp string) string {
	if len(str1) == 0 {
		return str2
	}
	if len(str2) == 0 {
		return str1
	}

	return Format(`%s%s%s`, str1, sp, str2)
}

/**
* AppendAny
* @param val1 interface{}
* @param val2 interface{}
* @param sp string
* @return interface{}
**/
func AppendAny(val1, val2 interface{}, sp string) interface{} {
	str1 := fmt.Sprintf(`%v`, val1)
	str2 := fmt.Sprintf(`%v`, val2)

	if len(str1) == 0 {
		return val2
	}
	if len(str2) == 0 {
		return val1
	}

	return Format(`%v%s%v`, val1, sp, val2)
}

/**
* Split
* @param str string
* @param sep string
* @return []string
**/
func Split(str, sep string) []string {
	return strings.Split(str, sep)
}

/**
* GetSplitIndex
* @param str string
* @param sep string
* @param idx int
* @return string
**/
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

/**
* ApendAny
* @param space string
* @param args ...any
* @return string
**/
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

/**
* StrToTime
* @param val string
* @return time.Time, error
**/
func StrToTime(val string) (time.Time, error) {
	var result time.Time
	layout := "2006-01-02T15:04:05.000Z"

	result, err := time.Parse(layout, val)
	if err != nil {
		return result, err
	}

	return result, nil
}

/**
* StrToBool
* @param val string
* @return bool, error
**/
func StrToBool(val string) (bool, error) {
	if Lowcase(val) == "true" {
		return true, nil
	} else if Lowcase(val) == "false" {
		return false, nil
	}

	return false, errors.New("invalid boolean value")
}

/**
* HtmlToText
* @param html string
* @return string
**/
func HtmlToText(html string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(html, "")
}

/**
* RemoveAcents
* @param str string
* @return string
**/
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
