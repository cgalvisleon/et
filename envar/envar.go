package envar

import (
	"os"

	"github.com/cgalvisleon/et/generic"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	_ "github.com/joho/godotenv/autoload"
)

func MetaSet(name string, _default any, usage, _var string) *generic.Any {
	var result *generic.Any = generic.New(_default)
	ok := false
	for _, arg := range os.Args[1:] {
		if ok {
			if arg == "" {
				logs.Errorf(`-%s in %s (default %s)`, name, usage, _default)
			}
			os.Setenv(_var, arg)
			result.Set(arg)
			break
		} else if arg == strs.Format(`-%s`, name) {
			ok = true
		}
	}

	return result
}

func SetvarAny(name string, _default any, usage, _var string) *generic.Any {
	result := MetaSet(name, _default, usage, _var)
	return result
}

func SetvarStr(name string, _default string, usage, _var string) string {
	result := MetaSet(name, _default, usage, _var)
	return result.Str()
}

func SetvarInt(name string, _default int, usage, _var string) int {
	result := MetaSet(name, _default, usage, _var)
	return result.Int()
}

func EnvarAny(arg string) *generic.Any {
	val := os.Getenv(arg)

	return generic.New(val)
}

func EnvarStr(_default string, arg string) string {
	result := EnvarAny(arg)

	if result.IsNil() {
		return _default
	}

	return result.Str()
}

func EnvarInt(_default int, arg string) int {
	result := EnvarAny(arg)

	if result.IsNil() {
		return _default
	}

	return result.Int()
}
