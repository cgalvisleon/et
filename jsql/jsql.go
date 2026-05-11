package jsql

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

var dbs map[string]*DB

func init() {
	dbs = make(map[string]*DB)
}

func LoadTo(params utility.Config) (*DB, error) {
	name := params.GetStr("DB_NAME", "test")
	result, ok := dbs[name]
	if ok {
		return result, nil
	}

	result, err := newDB(params)
	if err != nil {
		return nil, err
	}

	err = result.init()
	if err != nil {
		return nil, err
	}

	dbs[name] = result
	return result, nil
}

/**
* Load loads the default database
* @return (*DB, error)
 */
func Load() (*DB, error) {
	config := utility.NewConfig(et.Json{
		"DB_DRIVER":       envar.GetStr("DB_DRIVER", "postgres"),
		"DB_NAME":         envar.GetStr("DB_NAME", "test"),
		"DB_HOST":         envar.GetStr("DB_HOST", "localhost"),
		"DB_PORT":         envar.GetInt("DB_PORT", 5432),
		"DB_USER":         envar.GetStr("DB_USER", "postgres"),
		"DB_PASSWORD":     envar.GetStr("DB_PASSWORD", "postgres"),
		"DB_RECORD_LIMIT": envar.GetInt("DB_RECORD_LIMIT", 1000),
	})
	return LoadTo(config)
}
