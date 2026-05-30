package jsql

import (
	"encoding/json"
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

type Instance struct {
	model *Model
}

/**
* DefineInstance
* @param db *DB, schema string, name string
* @return *Model, error
**/
func DefineInstance(db *DB, schema string, name string) (*Instance, error) {
	result, err := db.Define(Def{
		Schema:  schema,
		Name:    name,
		Version: 1,
		Columns: []Column{
			{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
			{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
			{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "tag", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "owner_id", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "definition", TypeColumn: COLUMN, TypeData: BYTES, Default: []byte{}},
		},
		PrimaryKeys: []DefIndex{
			{Name: ID, Sorted: true},
		},
		Indexes: []DefIndex{
			{Name: "tag", Sorted: true},
		},
		IdxField: IDX,
		IsCore:   true,
		IsDebug:  true,
	})
	if err != nil {
		return nil, err
	}
	err = result.Init()
	if err != nil {
		return nil, err
	}

	return &Instance{model: result}, nil
}

/**
* Set
* @param id, tag string, obj any
* @return error
**/
func (s *Instance) Set(id, tag string, obj any) error {
	if s.model == nil {
		return errors.New(MSG_INSTANCE_REQUIRED_ID)
	}

	bt, ok := obj.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(obj)
		if err != nil {
			return err
		}
	}

	now := utility.Now()
	_, err := s.model.
		Upsert(et.Json{
			ID:           id,
			"tag":        tag,
			"definition": bt,
		}).
		BeforeInsert(func(tx *Tx, old, new et.Json) error {
			new[CREATED_AT] = now
			new[UPDATED_AT] = now
			return nil
		}).
		BeforeUpdate(func(tx *Tx, old, new et.Json) error {
			new[UPDATED_AT] = now
			return nil
		}).
		Exec()
	if err != nil {
		return err
	}

	return nil
}

/**
* Get
* @param id string, dest any
* @return (bool, error)
**/
func (s *Instance) Get(id string, dest any) (bool, error) {
	if s.model == nil {
		return false, errors.New(MSG_INSTANCE_REQUIRED_ID)
	}

	items, err := s.model.
		Where(Eq(ID, id)).
		One()
	if err != nil {
		return false, err
	}

	if !items.Ok {
		return false, nil
	}

	scr, err := items.Byte("definition")
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(scr, dest)
	if err != nil {
		return false, err
	}

	return true, nil
}

/**
* Delete
* @param id string
* @return error
**/
func (s *Instance) Delete(id string) error {
	if s.model == nil {
		return nil
	}

	_, err := s.model.
		Delete().
		Where(Eq(ID, id)).
		Exec()
	if err != nil {
		return err
	}

	return nil
}

/**
* Query
* @param query et.Json
* @return (et.Items, error)
**/
func (s *Instance) Query(query et.Json) (et.Items, error) {
	if s.model == nil {
		return et.Items{}, errors.New(MSG_INSTANCE_REQUIRED_ID)
	}

	return s.model.Query(query)
}
