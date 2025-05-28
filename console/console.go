package console

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/stdrout"
)

func printLn(kind string, color string, args ...any) {
	stdrout.Printl(kind, color, args...)

	event.Publish("logs", et.Json{
		"kind":    kind,
		"message": fmt.Sprint(args...),
	})
}

/**
* Log
* @param kind string, args ...any
* @return error
**/
func Log(kind string, args ...any) error {
	printLn(kind, "", args...)
	return nil
}

/**
* Logf
* @param kind string, format string, args ...any
* @return error
**/
func Logf(kind string, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	Log(kind, message)
}

/**
* Alert
* @param err error
* @return error
**/
func Alert(err error) error {
	if err != nil {
		printLn("Alert", "Yellow", err.Error())
	}

	return err
}

/**
* Alertm
* @param message string
* @return error
**/
func Alertm(message string) error {
	err := errors.New(message)
	if err != nil {
		Alert(err)
	}

	return err
}

/**
* Alertf
* @param format string, args ...any
* @return error
**/
func Alertf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	return Alert(err)
}

/**
* Tracer
* @param kind string, color string, err error
* @return error
**/
func Tracer(kind, color string, err error) error {
	if err == nil {
		return nil
	}

	return stdrout.Traces(kind, color, err)
}

/**
* Error
* @param err error
* @return error
**/
func Error(err error) error {
	functionName := stdrout.PrintFunctionName()
	if err != nil {
		printLn("Error", "Yellow", err.Error(), " - ", functionName)
	}

	return err
}

/**
* Errorm
* @param message string
* @return error
**/
func Errorm(message string) error {
	err := errors.New(message)
	return Error(err)
}

/**
* Errorf
* @param format string, args ...any
* @return error
**/
func Errorf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	err := errors.New(message)
	return Error(err)
}

/**
* Info
* @param v ...any
**/
func Info(v ...any) {
	printLn("Info", "Blue", v...)
}

/**
* Infof
* @param format string, args ...any
**/
func Infof(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	Info(message)
}

/**
* Stop
**/
func Stop() {
	Log("STOP")
	os.Exit(0)
}

/**
* Fatal
* @param err error
* @return error
**/
func Fatal(err error) error {
	Tracer("Fatal", "Red", err)
	os.Exit(1)

	return err
}

/**
* Fatalf
* @param format string, args ...any
* @return error
**/
func Fatalf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	err := errors.New(message)
	return Fatal(err)
}

/**
* Fatalm
* @param message string
* @return error
**/
func Fatalm(message string) error {
	err := errors.New(message)
	return Fatal(err)
}

/**
* Panic
* @param err error
* @return error
**/
func Panic(err error) error {
	functionName := stdrout.PrintFunctionName()
	if err != nil {
		printLn("Panic", "Red", err.Error(), " - ", functionName)
	}
	os.Exit(1)

	return err
}

/**
* Panicf
* @param format string, args ...any
* @return error
**/
func Panicf(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	err := errors.New(message)
	return Panic(err)
}

/**
* Ping
**/
func Ping() {
	Log("PING", "PONG")
}

/**
* Pong
**/
func Pong() {
	Log("PoNG", "PING")
}

/**
* Debug
* @param v ...any
**/
func Debug(v ...any) {
	production := config.App.Production
	if production {
		return
	}

	printLn("Debug", "Cyan", v...)
}

/**
* Debugf
* @param format string, args ...any
**/
func Debugf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	Debug(message)
}

/**
* Rpc
* @param packageName string, args string
* @return error
**/
func Rpc(packageName, args string) error {
	pc, _, _, _ := runtime.Caller(1)
	fullFuncName := runtime.FuncForPC(pc).Name()
	funcName := fullFuncName[strings.LastIndex(fullFuncName, "/")+1:]
	funcName = funcName[strings.LastIndex(funcName, ".")+1:]
	message := append([]any{packageName, ".", funcName, ":"}, args)
	printLn("Rpc", "Blue", message...)

	return nil
}

/**
* QueryError
* @param err error, sql string
* @return error
**/
func QueryError(err error, sql string) error {
	if err != nil {
		printLn("QueryError", "Red", err.Error(), " - SQL: ", sql)
	}

	return err
}
