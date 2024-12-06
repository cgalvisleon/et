package utility

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/cgalvisleon/et/stdrout"
	"golang.org/x/exp/slices"
)

func NewError(message string) error {
	return errors.New(message)
}

func NewErrorf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	return NewError(message)
}

func PrintFunctionName() string {
	pc, _, _, _ := runtime.Caller(2)
	fullFuncName := runtime.FuncForPC(pc).Name()

	return fullFuncName
}

func Traces(kind, color string, err error) ([]string, error) {
	var n int = 1
	var traces []string = []string{err.Error()}

	stdrout.Printl(kind, color, err.Error())

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
			stdrout.Printl("TRACE", color, trace)
		}
	}

	return traces, err
}
