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
* @param format string, args ...any
* @return string
**/
func Format(format string, args ...any) string {
	result := fmt.Sprintf(format, args...)

	return result
}

/**
* FormatUppCase
* @param format string, args ...any
* @return string
**/
func FormatUppCase(format string, args ...any) string {
	result := Format(format, args...)

	return Uppcase(result)
}

/**
* FormatLowCase
* @param format string, args ...any
* @return string
**/
func FormatLowCase(format string, args ...any) string {
	result := Format(format, args...)

	return Lowcase(result)
}

/**
* FormatDateTime
* @param format string, value time.Time
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
* FormatSerie
* @param format string, num int64
* @return string
**/
func FormatSerie(format string, num int64) string {
	re := regexp.MustCompile("0")
	format = re.ReplaceAllString(format, "%0")

	return Format(format, num)
}

/**
* Contains
* @param str string, substr string
* @return bool
**/
func Contains(str string, substr string) bool {
	return strings.Contains(str, substr)
}

/**
* Replace
* @param str string, old string, new string
* @return string
**/
func Replace(str string, old string, new string) string {
	return strings.ReplaceAll(str, old, new)
}

/**
* ReplaceAll
* @param str string, olds []string, new string
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
* Change
* @param str string, olds []string, news []string
* @return string
**/
func Change(str string, olds []string, news []string) string {
	var result = str
	for i, s := range olds {
		old := Format(`\b%s\b`, s)
		new := news[i]
		re := regexp.MustCompile(old)
		result = re.ReplaceAllString(result, new)
		old = Format(`\b%s\b`, Uppcase(s))
		new = Uppcase(news[i])
		re = regexp.MustCompile(old)
		result = re.ReplaceAllString(result, new)
		old = Format(`\b%s\b`, Lowcase(s))
		new = Lowcase(news[i])
		re = regexp.MustCompile(old)
		result = re.ReplaceAllString(result, new)
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
* Same
* @param str1 string, str2 string
* @return bool
**/
func Same(str1, str2 string) bool {
	return strings.EqualFold(str1, str2)
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
* IsEmpty
* @param str string
* @return bool
**/
func IsEmpty(str string) bool {
	return len(str) == 0
}

/**
* Append
* @param str1 string, str2 string, sp string
* @return string
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
* @param val1 interface{}, val2 interface{}, sp string
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
* @param str string, sep string
* @return []string
**/
func Split(str, sep string) []string {
	return strings.Split(str, sep)
}

/**
* GetSplitIndex
* @param str string, sep string, idx int
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
* @param space string, args ...any
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
