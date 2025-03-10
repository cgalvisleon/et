package console

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/stdrout"
)

func printLn(kind string, color string, args ...any) {
	stdrout.Printl(kind, color, args...)

	event.Publish("logs", et.Json{
		"kind":    kind,
		"message": fmt.Sprint(args...),
	})
}

func Log(kind string, args ...any) error {
	printLn(kind, "", args...)
	return nil
}

func Logf(kind string, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	Log(kind, message)
}

func Alert(err error) error {
	if err != nil {
		printLn("Alert", "Yellow", err.Error())
	}

	return err
}

func Alertm(message string) error {
	err := mistake.New(message)
	if err != nil {
		Alert(err)
	}

	return err
}

func Alertf(format string, args ...any) error {
	err := mistake.Newf(format, args...)
	return Alert(err)
}

func Tracer(kind, color string, err error) error {
	if err == nil {
		return nil
	}

	return stdrout.Traces(kind, color, err)
}

func Error(err error) error {
	functionName := stdrout.PrintFunctionName()
	if err != nil {
		printLn("Error", "Yellow", err.Error(), " - ", functionName)
	}

	return err
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
	printLn("Info", "Blue", v...)
}

func Infof(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	Info(message)
}

func Stop() {
	Log("STOP")
	os.Exit(0)
}

func Fatal(err error) error {
	Tracer("Fatal", "Red", err)
	os.Exit(1)

	return err
}

func Fatalm(message string) error {
	err := mistake.New(message)
	return Fatal(err)
}

func Panic(err error) error {
	functionName := stdrout.PrintFunctionName()
	if err != nil {
		printLn("Panic", "Red", err.Error(), " - ", functionName)
	}
	os.Exit(1)

	return err
}

func Panicf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	err := mistake.New(message)
	return Panic(err)
}

func Ping() {
	Log("PING", "PONG")
}

func Pong() {
	Log("PoNG", "PING")
}

func Debug(v ...any) {
	production := envar.Bool("PRODUCTION")
	if production {
		return
	}

	printLn("Debug", "Cyan", v...)
}

func Debugf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	Debug(message)
}

func Rpc(args ...any) error {
	pc, _, _, _ := runtime.Caller(1)
	fullFuncName := runtime.FuncForPC(pc).Name()
	funcName := fullFuncName[strings.LastIndex(fullFuncName, "/")+1:]
	message := append([]any{funcName, ":"}, args...)
	printLn("Rpc", "Blue", message...)

	return nil
}

func QueryError(err error, sql string) error {
	if err != nil {
		printLn("QueryError", "Red", err.Error(), " - SQL: ", sql)
	}

	return err
}
