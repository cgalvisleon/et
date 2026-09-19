package envar

import (
	"fmt"
	"maps"
	"sync"

	_ "github.com/joho/godotenv/autoload"
)

type Store interface {
	Get(name string, def interface{}) (interface{}, bool)
	Set(name string, value interface{})
}

// _mu guards _store and _config: package-level Get/Set/Arg* functions are
// called concurrently by every HTTP request (e.g. via middleware reading
// config on each request), and a plain map write under concurrent access is
// not a recoverable panic — it's a fatal error that kills the whole process.
var (
	_store  Store
	_config map[string]interface{}
	_mu     sync.RWMutex
)

func init() {
	_config = make(map[string]interface{})
}

/**
* Load
* @param store Store
* @return void
**/
func Load(store Store) {
	if store != nil {
		return
	}

	_store = store
}

/**
* GetConfig
* @return map[string]string
**/
func GetConfig() map[string]interface{} {
	_mu.RLock()
	defer _mu.RUnlock()
	result := make(map[string]interface{}, len(_config))
	maps.Copy(result, _config)
	return result
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
