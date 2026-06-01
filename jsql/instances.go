package jsql

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
)

type Store interface {
	Set(id, tag, ownerId string, obj any) error
	Get(id string, dest any) (bool, error)
	Delete(id string) error
	Query(query et.Json) (et.Items, error)
}

type Kind string

const (
	KindJson Kind = "json"
	KindBite Kind = "binary"
)

type Instance struct {
	model *Model
	kind  Kind
}

var instance *Instance

/**
* NewInstance
* @param db *DB, schema, name string, kind Kind
* @return (*Instance, error)
**/
func NewInstance(db *DB, schema, name string, kind Kind) (*Instance, error) {
	columns := []Column{
		{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "tag", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "title", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
		{Name: "owner_id", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: SOURCE, TypeColumn: COLUMN, TypeData: BYTES, Default: []byte{}},
	}

	def := Def{
		Schema:  schema,
		Name:    name,
		Version: 1,
		Columns: columns,
		PrimaryKeys: []DefIndex{
			{Name: ID, Sorted: true},
		},
		Indexes: []DefIndex{
			{Name: "tag", Sorted: true},
			{Name: "title", Sorted: true},
			{Name: "owner_id", Sorted: true},
		},
		IdxField: IDX,
		IdtField: IDT,
		IsCore:   true,
		IsDebug:  true,
	}
	if kind == KindJson {
		def.Columns[5].TypeData = JSON
		def.SourceField = SOURCE
	}

	result, err := db.Define(def)
	if err != nil {
		return nil, err
	}
	result.BeforeInsert(func(tx *Tx, old, new et.Json) error {
		now := timezone.Now()
		new.Set(CREATED_AT, now)
		new.Set(UPDATED_AT, now)
		return nil
	})
	result.BeforeUpdate(func(tx *Tx, old, new et.Json) error {
		now := timezone.Now()
		new.Set(UPDATED_AT, now)
		return nil
	})
	err = result.Init()
	if err != nil {
		return nil, err
	}

	return &Instance{model: result}, nil
}

/**
* LoadInstance
* @param db *DB, schema, name string, kind Kind
* @return (*Instance, error)
**/
func LoadInstance(db *DB, schema, name string, kind Kind) (*Instance, error) {
	if instance != nil {
		return instance, nil
	}
	instance, err := NewInstance(db, schema, name, kind)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

/**
* Set
* @param id, tag, ownerId string, obj any
* @return error
**/
func (s *Instance) Set(id, tag, ownerId string, obj any) error {
	bt, ok := obj.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(obj)
		if err != nil {
			return err
		}
	}

	var data = et.Json{}
	if s.kind == KindJson {
		err := json.Unmarshal(bt, &data)
		if err != nil {
			return err
		}
	} else {
		data = et.Json{
			SOURCE: bt,
		}
	}

	data.Set(ID, id)
	data.Set("tag", tag)
	data.Set("owner_id", ownerId)
	_, err := s.model.
		Upsert(data).
		Where(Eq(ID, id)).
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
	item, err := s.model.
		Where(Eq(ID, id)).
		One()
	if err != nil {
		return false, err
	}

	if !item.Ok {
		return false, nil
	}

	bt := []byte(item.Result.ToString())
	err = json.Unmarshal(bt, dest)
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
	return s.model.Query(query)
}
