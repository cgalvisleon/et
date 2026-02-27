package logs

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cgalvisleon/et/stdrout"
)

/**
* printLn
* @param kind string, color string, args ...any
* @return string
**/
func printLn(kind string, color string, args ...any) {
	stdrout.Printl(kind, color, args...)
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
* @param kind string, err error
* @return error
**/
func Tracer(kind string, err error) error {
	if err == nil {
		return nil
	}

	return stdrout.Traces(kind, "Red", err)
}

/**
* Error
* @param err error
* @return error
**/
func Error(err error) error {
	functionName := stdrout.GetFunctionName(3)
	if err != nil {
		printLn("Error", "Red", err.Error(), " - ", functionName)
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
* Fatal
* @param err error
* @return error
**/
func Fatal(err error) error {
	Tracer("Fatal", err)
	os.Exit(1)

	return err
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
	printLn("Panic", "Red", err)
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
	printLn("Debug", "Cyan", v...)
}

/**
* Debugger
* @param v ...any
**/
func Debugger(v ...any) {
	printLn("Debugger", "Cyan", v...)
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
* @param args ...any
* @return error
**/
func Rpc(args ...any) error {
	fullFuncName := stdrout.GetFunctionName(1)
	funcName := fullFuncName[strings.LastIndex(fullFuncName, "/")+1:]
	message := append([]any{funcName, ":"}, args...)
	printLn("Rpc", "Blue", message...)

	return nil
}
