package envar

import (
	"fmt"

	_ "github.com/joho/godotenv/autoload"
)

type Store interface {
	Get(name string, def interface{}) interface{}
}

var (
	_store Store
)

/**
* Load
* @param store Store
* @return void
**/
func Load(store Store) {
	_store = store
}

/**
* Validate
* @param keys []string
* @return error
**/
func Validate(keys []string) error {
	for _, key := range keys {
		val := Get(key, "")
		if val == "" {
			return fmt.Errorf(MSG_ATRIB_REQUIRED, key)
		}
	}
	return nil
}
