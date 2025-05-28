package logs

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/cgalvisleon/et/stdrout"
)

func printLn(kind string, color string, args ...any) {
	stdrout.Printl(kind, color, args...)
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
	err := errors.New(message)
	if err != nil {
		Alert(err)
	}

	return err
}

func Alertf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)
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
	err := errors.New(message)
	return Error(err)
}

func Errorf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	err := errors.New(message)
	return Error(err)
}

func Info(v ...any) {
	printLn("Info", "Blue", v...)
}

func Infof(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	Info(message)
}

func Fatal(err error) error {
	Tracer("Fatal", "Red", err)
	os.Exit(1)

	return err
}

func Fatalm(message string) error {
	err := errors.New(message)
	return Fatal(err)
}

func Panic(err error) error {
	printLn("Panic", "Red", err)
	os.Exit(1)

	return err
}

func Panicf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	err := errors.New(message)
	return Panic(err)
}

func Ping() {
	Log("PING", "PONG")
}

func Pong() {
	Log("PoNG", "PING")
}

func Debug(v ...any) {
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
