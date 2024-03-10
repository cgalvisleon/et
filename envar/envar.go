package envar

import (
	"fmt"
	"os"
	"strconv"
)

func MetaSet(name string, _default any, usage, _var string) string {
	var result string
	ok := false
	for _, arg := range os.Args[1:] {
		if ok {
			os.Setenv(_var, arg)
			result = arg
			break
		} else if arg == fmt.Sprintf(`-%s`, name) {
			ok = true
		}
	}

	return result
}

func SetvarStr(name string, _default string, usage, _var string) string {
	result := MetaSet(name, _default, usage, _var)
	return result
}

func SetvarInt(name string, _default int, usage, _var string) int {
	str := MetaSet(name, _default, usage, _var)
	result, err := strconv.Atoi(str)
	if err != nil {
		return _default
	}

	return result
}

func EnvarStr(_default string, arg string) string {
	val := os.Getenv(arg)
	if val == "" {
		return _default
	}

	return val
}

func EnvarInt(_default int, arg string) int {
	val := os.Getenv(arg)
	if val == "" {
		return _default
	}

	result, err := strconv.Atoi(val)
	if err != nil {
		return _default
	}

	return result
}
