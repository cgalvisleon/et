package envar

import (
	"os"
	"strconv"

	"github.com/cgalvisleon/et/strs"
	_ "github.com/joho/godotenv/autoload"
)

func MetaSet(name string, _default string, description, _var string) string {
	for i, arg := range os.Args[1:] {
		if arg == strs.Format("-%s", name) {
			val := os.Args[i+2]
			os.Setenv(_var, val)
			return val
		}
	}

	return _default
}

func SetvarStr(name string, _default string, usage, _var string) string {
	return MetaSet(name, _default, usage, _var)
}

func SetvarInt(name string, _default int, usage, _var string) int {
	result := MetaSet(name, strconv.Itoa(_default), usage, _var)

	val, err := strconv.Atoi(result)
	if err != nil {
		return _default
	}

	return val
}

func SetvarBool(name string, _default bool, usage, _var string) bool {
	result := MetaSet(name, strconv.FormatBool(_default), usage, _var)

	val, err := strconv.ParseBool(result)
	if err != nil {
		return _default
	}

	return val
}

func EnvarStr(_default string, _var string) string {
	result := os.Getenv(_var)

	if result == "" {
		return _default
	}

	return result
}

func EnvarInt(_default int, _var string) int {
	result := EnvarStr(strconv.Itoa(_default), _var)

	val, err := strconv.Atoi(result)
	if err != nil {
		return _default
	}

	return val
}

func EnvarBool(_default bool, _var string) bool {
	result := EnvarStr(strconv.FormatBool(_default), _var)

	val, err := strconv.ParseBool(result)
	if err != nil {
		return _default
	}

	return val
}
