package logs

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/cgalvisleon/et/strs"
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
	}
}

func Logln(kind string, color string, args ...any) string {
	kind = strings.ToUpper(kind)
	message := fmt.Sprint(args...)
	now := time.Now().Format("2006/01/02 15:04:05")
	var result string

	switch color {
	case "Reset":
		result = now + Purple + strs.Format(" [%s]: ", kind) + Reset + message + Reset
	case "Red":
		result = now + Purple + strs.Format(" [%s]: ", kind) + Reset + Red + message + Reset
	case "Green":
		result = now + Purple + strs.Format(" [%s]: ", kind) + Reset + Green + message + Reset
	case "Yellow":
		result = now + Purple + strs.Format(" [%s]: ", kind) + Reset + Yellow + message + Reset
	case "Blue":
		result = now + Purple + strs.Format(" [%s]: ", kind) + Reset + Blue + message + Reset
	case "Purple":
		result = now + Purple + strs.Format(" [%s]: ", kind) + Reset + Purple + message + Reset
	case "Cyan":
		result = now + Purple + strs.Format(" [%s]: ", kind) + Reset + Cyan + message + Reset
	case "Gray":
		result = now + Purple + strs.Format(" [%s]: ", kind) + Reset + Gray + message + Reset
	case "White":
		result = now + Purple + strs.Format(" [%s]: ", kind) + Reset + White + message + Reset
	default:
		result = now + Purple + strs.Format(" [%s]: ", kind) + Reset + Green + message + Reset
	}

	println(result)

	return result
}

func Log(kind string, args ...any) {
	Logln(kind, "", args...)
}

func Logf(kind string, format string, args ...any) {
	message := strs.Format(format, args...)
	Logln(kind, "", message)
}

func Error(err error) error {
	Logln("ERROR", "Red", err.Error())
	return err
}

func Errorm(message string) error {
	err := errors.New(message)
	return Error(err)
}

func Errorf(format string, args ...any) error {
	message := strs.Format(format, args...)
	err := errors.New(message)
	return Error(err)
}

func Fatal(v ...any) {
	Logln("Fatal", "Red", v...)
	os.Exit(1)
}

func Panic(v ...any) {
	Logln("Panic", "Red", v...)
	os.Exit(1)
}
