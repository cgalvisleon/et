package envar

import (
	"os"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/strs"
	_ "github.com/joho/godotenv/autoload"
)

func metaSet(name, def, description, _var string) string {
	for i, arg := range os.Args[1:] {
		if arg == strs.Format("-%s", name) {
			val := os.Args[i+2]
			os.Setenv(_var, val)
			return val
		}
	}

	return def
}

// Set a string environment variable
func SetStr(name, def, description, _var string) string {
	return metaSet(name, def, description, _var)
}

// Set an integer environment variable
func SetInt(name string, def int, description, _var string) int {
	result := metaSet(name, strconv.Itoa(def), description, _var)

	val, err := strconv.Atoi(result)
	if err != nil {
		return def
	}

	return val
}

// Set a boolean environment variable
func SetBool(name string, def bool, description, _var string) bool {
	result := metaSet(name, strconv.FormatBool(def), description, _var)

	val, err := strconv.ParseBool(result)
	if err != nil {
		return def
	}

	return val
}

// Set a time environment variable
func SetTime(name string, def time.Time, description, _var string) time.Time {
	result := metaSet(name, def.Format(time.RFC3339), description, _var)

	val, err := time.Parse(time.RFC3339, result)
	if err != nil {
		return def
	}

	return val
}

// Get a string environment variable
func GetStr(def string, _var string) string {
	result := os.Getenv(_var)

	if result == "" {
		return def
	}

	return result
}

// Get an integer environment variable
func GetInt(def int, _var string) int {
	result := GetStr(strconv.Itoa(def), _var)

	val, err := strconv.Atoi(result)
	if err != nil {
		return def
	}

	return val
}

func GetInt64(def int64, _var string) int64 {
	result := GetStr(strconv.FormatInt(def, 10), _var)

	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return def
	}

	return val
}

// Get a boolean environment variable
func GetBool(def bool, _var string) bool {
	result := GetStr(strconv.FormatBool(def), _var)

	val, err := strconv.ParseBool(result)
	if err != nil {
		return def
	}

	return val
}

// Get a time environment variable
func GetTime(def time.Time, _var string) time.Time {
	result := GetStr(def.Format(time.RFC3339), _var)

	val, err := time.Parse(time.RFC3339, result)
	if err != nil {
		return def
	}

	return val
}
