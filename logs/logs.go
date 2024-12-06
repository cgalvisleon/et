package logs

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/timezone"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Purple = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"
var useColor = true

func init() {
	if runtime.GOOS == "windows" {
		Reset = ""
		Red = ""
		Green = ""
		Yellow = ""
		Blue = ""
		Purple = ""
		Cyan = ""
		Gray = ""
		White = ""
		useColor = false
	}
}

func log(kind string, color string, args ...any) string {
	now := timezone.Now()
	kind = strings.ToUpper(kind)
	message := fmt.Sprint(args...)
	var result string

	switch color {
	case "Reset":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + message + Reset
	case "Red":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Red + message + Reset
	case "Green":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Green + message + Reset
	case "Yellow":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Yellow + message + Reset
	case "Blue":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Blue + message + Reset
	case "Purple":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Purple + message + Reset
	case "Cyan":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Cyan + message + Reset
	case "Gray":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Gray + message + Reset
	case "White":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + White + message + Reset
	default:
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Green + message + Reset
	}

	println(result)

	return result
}

var ping = 0

func Log(kind string, args ...any) error {
	log(kind, "", args...)
	return nil
}

func Logf(kind string, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	log(kind, "", message)
}

func Traces(kind, color string, err error) error {
	log(kind, color, err.Error())

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
			log("TRACE", color, trace)
		}
	}

	return err
}

func errorTraces(err error) []string {
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

func errorTarget(kind, color string, err error) error {
	traces := errorTraces(err)
	if len(traces) > 0 {
		n := len(traces) - 1
		log(kind, color, err.Error(), "::", traces[n])

		return err
	}

	log(kind, color, err.Error())

	return err
}

func PrintFunctionName() string {
	pc, _, _, _ := runtime.Caller(2)
	fullFuncName := runtime.FuncForPC(pc).Name()

	return fullFuncName
}

func Alert(err error) error {
	if err != nil {
		log("Alert", "Yellow", err.Error())
	}

	return err
}

func Alertm(message string) error {
	err := mistake.New(message)
	if err != nil {
		log("Alert", "Yellow", err.Error())
	}

	return err
}

func Alertf(format string, args ...any) error {
	err := mistake.Newf(format, args...)
	return Alert(err)
}

func AlertT(err error) error {
	functionName := PrintFunctionName()
	if err != nil {
		log("Alert", "Yellow", err.Error(), " - ", functionName)
	}

	return err
}

func Trace(err error) error {
	if err == nil {
		return nil
	}

	return Traces("Trace", "Blue", err)
}

func Error(err error) error {
	return Traces("Error", "Red", err)
}

func Errorm(message string) error {
	err := mistake.New(message)
	return Error(err)
}

func Errorf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	err := mistake.New(message)
	return Error(err)
}

func Info(v ...any) {
	log("Info", "Blue", v...)
}

func Infof(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	log("Info", "Blue", message)
}

func Fatal(err error) error {
	Traces("Fatal", "Red", err)
	os.Exit(1)

	return err
}

func Fatalm(message string) error {
	err := mistake.New(message)
	return Fatal(err)
}

func Panic(err error) error {
	Traces("Panic", "Red", err)
	os.Exit(1)

	return err
}

func Panicf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	err := mistake.New(message)
	return Panic(err)
}

func Ping() {
	ping++
	Log("PONG", ping)
}

func Debug(v ...any) {
	log("Debug", "Cyan", v...)
}

func Debugf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	log("Debug", "Cyan", message)
}

func Rpc(args ...any) error {
	pc, _, _, _ := runtime.Caller(1)
	fullFuncName := runtime.FuncForPC(pc).Name()
	funcName := fullFuncName[strings.LastIndex(fullFuncName, "/")+1:]
	message := append([]any{funcName, ":"}, args...)
	log("Rpc", "Blue", message...)

	return nil
}
