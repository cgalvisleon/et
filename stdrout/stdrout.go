package stdrout

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/timezone"
)

var (
	Reset  = "\033[97m"
	Black  = "\033[30m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	White  = "\033[97m"
	// Bytes
	BBlack   = []byte{'\033', '[', '3', '0', ';', '1', 'm'}
	BRed     = []byte{'\033', '[', '3', '1', ';', '1', 'm'}
	BGreen   = []byte{'\033', '[', '3', '2', ';', '1', 'm'}
	BYellow  = []byte{'\033', '[', '3', '3', ';', '1', 'm'}
	BBlue    = []byte{'\033', '[', '3', '4', ';', '1', 'm'}
	BPurple  = []byte{'\033', '[', '3', '5', ';', '1', 'm'}
	BCyan    = []byte{'\033', '[', '3', '6', ';', '1', 'm'}
	BWhite   = []byte{'\033', '[', '3', '7', ';', '1', 'm'}
	BReset   = []byte{'\033', '[', '9', '7', 'm'}
	IsTTY    bool
	useColor = true
	colors   = map[string]string{
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
)

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

	fi, err := os.Stdout.Stat()
	if err == nil {
		m := os.ModeDevice | os.ModeCharDevice
		IsTTY = fi.Mode()&m == m
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
* CW
* @param w io.Writer, color []byte, format string, args ...interface{}
* @return string
**/
func CW(w io.Writer, color []byte, format string, args ...interface{}) {
	if IsTTY && useColor {
		w.Write(color)
	}
	fmt.Fprintf(w, format, args...)
	if IsTTY && useColor {
		w.Write([]byte(Reset))
	}
}

/**
* Printl
* @param kind string, color string, args ...any
* @return string
**/
func Printl(kind string, color string, args ...any) string {
	kind = strings.ToUpper(kind)
	message := fmt.Sprint(args...)
	now := timezone.NowStr()
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
* getFunctionName
* @return string
**/
func GetFunctionName(idx int) string {
	pc, _, _, ok := runtime.Caller(idx)
	if !ok {
		idx--
		return GetFunctionName(idx)
	}
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
