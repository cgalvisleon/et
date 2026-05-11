package jql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
)

type DB struct {
	Name      string             `json:"name"`
	Schemas   map[string]*Schema `json:"schemas"`
	UseCore   bool               `json:"use_core"`
	IsDebug   bool               `json:"-"`
	IsChanged bool               `json:"-"`
	driver    Driver             `json:"-"`
	db        *sql.DB            `json:"-"`
}

/**
* serialize
* @return []byte, error
**/
func (s *DB) serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *DB) ToJson() et.Json {
	bt, err := s.serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* Save
* @return error
**/
func (s *DB) Save() error {
	return nil
}

/**
* init
* @return error
**/
func (s *DB) init() error {
	if s.db != nil {
		return nil
	}

	if s.driver == nil {
		return errors.New(msg.MSG_DRIVER_NOT_FOUND)
	}

	db, err := s.driver.Connect(s)
	if err != nil {
		return err
	}

	s.db = db
	if s.UseCore {
		err := s.initCore()
		if err != nil {
			return err
		}
	}

	if s.IsChanged {
		return s.Save()
	}

	return nil
}

/**
* NewModel
* @param schema, name string, version int
* @return *Model
**/
func (s *DB) NewModel(schema, name string, version int) (*Model, error) {
	schema = utility.Normalize(schema)
	sch, ok := s.Schemas[schema]
	if !ok {
		sch = &Schema{
			Database: s.Name,
			Name:     schema,
			models:   make(map[string]*Model),
			db:       s,
			mu:       &sync.RWMutex{},
		}
		s.Schemas[schema] = sch
	}

	result, err := sch.newModel(name, version)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* SetDebug
* @param debug bool
**/
func (s *DB) SetDebug(debug bool) {
	s.IsDebug = debug
}

/**
* Debug
**/
func (s *DB) Debug() {
	s.IsDebug = true
}

/**
* getSchema
* @param name string
* @return (*Schema, error)
**/
func (s *DB) getSchema(name string) (*Schema, error) {
	result, ok := s.Schemas[name]
	if ok {
		return result, nil
	}

	return nil, fmt.Errorf(msg.MSG_SCHEMA_NOT_FOUND, name)
}

/**
* GetModel
* @param schema string, name string
* @return *Model
**/
func (s *DB) GetModel(schema string, name string) (*Model, error) {
	sch, err := s.getSchema(schema)
	if err != nil {
		return nil, err
	}

	result, err := sch.GetModel(name)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* sqlTx
* @param tx *Tx, sql string, arg ...any
* @return et.Items, error
*
 */
func (s *DB) sqlTx(tx *Tx, query string, arg ...any) (et.Items, error) {
	query = SQLParse(query, arg...)
	if tx != nil {
		err := tx.begin(s.db)
		if err != nil {
			return et.Items{}, err
		}

		rows, err := tx.Tx.Query(query)
		if err != nil {
			errR := tx.rollback()
			if errR != nil {
				err = fmt.Errorf(msg.MSG_ROLLBACK_ERROR, errR)
			}
			return et.Items{}, err
		}
		result := RowsToItems(rows)
		return result, nil
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return et.Items{}, err
	}

	result := RowsToItems(rows)
	return result, nil
}

/**
* Load
* @param model *Model
* @return error
**/
func (s *DB) load(model *Model) error {
	if s.driver == nil {
		return errors.New(msg.MSG_DRIVER_NOT_FOUND)
	}

	sql, err := s.driver.Load(model)
	if err != nil {
		return err
	}

	_, err = s.sqlTx(nil, sql)
	if err != nil {
		return err
	}

	return nil
}

/**
* Command
* @param command *Command
* @return string, error
**/
func (s *DB) command(command *Command) (string, error) {
	if s.driver == nil {
		return "", errors.New(msg.MSG_DRIVER_NOT_FOUND)
	}

	if s.IsDebug {
		logs.Debugf("command:%s", command.ToJson().ToEscapeHTML())
	}

	return s.driver.Command(command)
}

/**
* Query
* @param query *Query
* @return string, error
**/
func (s *DB) query(query *Query) (string, error) {
	if s.driver == nil {
		return "", errors.New(msg.MSG_DRIVER_NOT_FOUND)
	}

	if s.IsDebug {
		logs.Debugf("query:%s", query.ToJson().ToEscapeHTML())
	}

	return s.driver.Query(query)
}

/**
* Define
* @param definition Define
* @return *Model, error
**/
func (s *DB) Define(definition Define) (*Model, error) {
	return nil, nil
}
