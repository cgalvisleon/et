package jsql

import (
	"errors"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

var (
	dbs                    map[string]*DB
	ErrRecordAlreadyExists = errors.New("record already exists")
)

func init() {
	dbs = make(map[string]*DB)
}

/**
* LoadTo: Returns an existing DB by name, or creates and initialises a new one from params.
* @param connect Connection
* @return *DB, error
**/
func LoadTo(connect Connection) (*DB, error) {
	params := connect.GetParams()
	name := params.Str("database")
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
* Load: Connects to the default database reading configuration from environment variables.
* @return *DB, error
**/
func Load() (*DB, error) {
	config := PgConection{
		Database:    envar.GetStr("DB_NAME", "test"),
		Host:        envar.GetStr("DB_HOST", "localhost"),
		Port:        envar.GetInt("DB_PORT", 5432),
		User:        envar.GetStr("DB_USER", "test"),
		Password:    envar.GetStr("DB_PASSWORD", "test"),
		UseCore:     envar.GetBool("DB_USE_CORE", false),
		RecordLimit: envar.GetInt("DB_RECORD_LIMIT", 1000),
	}
	return LoadTo(&config)
}

/**
* GetDb: Returns an existing DB by name.
* @param name string
* @return *DB, error
**/
func GetDb(name string) (*DB, error) {
	db, ok := dbs[name]
	if !ok {
		return nil, errors.New(msg.MSG_DB_NOT_FOUND)
	}
	return db, nil
}

/**
* GetModel: Returns an existing Model by name.
* @param db string, schema string, name string
* @return *Model, error
**/
func GetModel(db string, schema string, name string) (*Model, error) {
	dbInstance, err := GetDb(db)
	if err != nil {
		return nil, err
	}

	model, err := dbInstance.GetModel(schema, name)
	if err != nil {
		return nil, err
	}
	return model, nil
}

/**
* NewModel: Creates a new Model by name.
* @param db *DB, schema string, name string, version int
* @return *Model, error
**/
func NewModel(db *DB, schema string, name string, version int) (*Model, error) {
	return db.NewModel(schema, name, version)
}

/**
* From: Creates a new Query with the specified model and optional alias.
* @param model *Model, as ...string
* @return *Query
**/
func From(model *Model, as ...string) *Query {
	asStr := ""
	if len(as) > 0 {
		asStr = as[0]
	}
	return newQuery(model, asStr)
}

/**
* Insert: Creates a new Insert command for the specified model with the given data.
* @param model *Model, data et.Json
* @return *Command
**/
func Insert(model *Model, data et.Json) *Command {
	return model.Insert(data)
}

/**
* Update: Creates a new Update command for the specified model with the given data.
* @param model *Model, data et.Json
* @return *Command
**/
func Update(model *Model, data et.Json) *Command {
	return model.Update(data)
}

/**
* Delete: Creates a new Delete command for the specified model.
* @param model *Model
* @return *Command
**/
func Delete(model *Model) *Command {
	return model.Delete()
}

/**
* Upsert: Creates a new Upsert command for the specified model with the given data.
* @param model *Model, data et.Json
* @return *Command
**/
func Upsert(model *Model, data et.Json) *Command {
	return model.Upsert(data)
}

/**
* Define: Creates a model from a declarative definition.
* @param dbName string, def Def
* @return *Model, error
**/
func Define(dbName string, def Def) (*Model, error) {
	db, err := GetDb(dbName)
	if err != nil {
		return nil, err
	}

	return db.Define(def)
}
