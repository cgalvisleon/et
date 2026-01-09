package stdrout

import (
	"fmt"
	"runtime"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/timezone"
)

var Reset = "\033[97m"
var Black = "\033[30m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Purple = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"
var colors = map[string]string{
	"Reset":  Reset,
	"Black":  Black,
	"Red":    Red,
	"Green":  Green,
	"Yellow": Yellow,
	"Blue":   Blue,
	"Purple": Purple,
	"Cyan":   Cyan,
	"Gray":   Gray,
	"White":  White,
	Reset:    Reset,
	Black:    Black,
	Red:      Red,
	Green:    Green,
	Yellow:   Yellow,
	Blue:     Blue,
	Purple:   Purple,
	Cyan:     Cyan,
	Gray:     Gray,
	White:    White,
}

func init() {
	if runtime.GOOS == "windows" {
		Reset = ""
		Black = ""
		Red = ""
		Green = ""
		Yellow = ""
		Blue = ""
		Purple = ""
		Cyan = ""
		Gray = ""
		White = ""
	}
}

/**
* Color
* @param s string, color string, format string, args ...interface{}
* @return string
**/
func Color(s *string, color string, format string, args ...interface{}) *string {
	result := ""
	printColor := colors[color]
	if printColor == Reset {
		result = Reset + fmt.Sprintf(format, args...)
	} else {
		result = Reset + printColor + fmt.Sprintf(format, args...)
	}

	if s == nil {
		return &result
	}

	if *s != "" {
		*s += result + Reset
	} else {
		*s = result + Reset
	}

	return s
}

/**
* Printl
* @param kind string, color string, args ...any
* @return string
**/
func Printl(kind string, color string, args ...any) string {
	kind = strings.ToUpper(kind)
	message := fmt.Sprint(args...)
	now := timezone.Now()
	var result string

	printColor := colors[color]
	if printColor == Reset {
		result = Reset + now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + message + Reset
	} else {
		result = Reset + now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + printColor + message + Reset
	}

	println(result)

	return result
}

/**
* Traces
* @param kind string, color string, err error
* @return error
**/
func Traces(kind, color string, err error) error {
	Printl(kind, color, err.Error())

	var n int = 1
	var traces []string = []string{err.Error()}
	for {
		pc, file, line, more := runtime.Caller(n)
		if !more {
			break
		}
		n++
		function := runtime.FuncForPC(pc)
		name := function.Name()
		list := strings.Split(name, ".")
		if len(list) > 0 {
			name = list[len(list)-1]
		}
		if !slices.Contains([]string{"ErrorM", "ErrorF"}, name) {
			trace := fmt.Sprintf("%s:%d func:%s", file, line, name)
			traces = append(traces, trace)
			Printl("TRACE", color, trace)
		}
	}

	return err
}

/**
* GetFunctionName
* @return string
**/
func GetFunctionName(idx int) string {
	pc, _, _, _ := runtime.Caller(idx)
	return runtime.FuncForPC(pc).Name()
}

/**
* ErrorTraces
* @param err error
* @return []string
**/
func ErrorTraces(err error) []string {
	var n = 1
	var traces = []string{err.Error()}
	for {
		pc, file, line, more := runtime.Caller(n)
		if !more {
			break
		}
		n++
		function := runtime.FuncForPC(pc)
		name := function.Name()
		list := strings.Split(name, ".")
		if len(list) > 0 {
			name = list[len(list)-1]
		}
		if !slices.Contains([]string{"ErrorM", "ErrorF"}, name) {
			trace := fmt.Sprintf("%s:%d func:%s", file, line, name)
			traces = append(traces, trace)
		}
	}

	return traces
}
