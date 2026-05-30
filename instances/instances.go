package instances

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/timezone"
)

type Kind string

const (
	KindJson Kind = "json"
	KindBite Kind = "binary"
)

type Instance struct {
	model *jsql.Model
	kind  Kind
}

var instance *Instance

/**
* New
* @param db *jsql.DB, schema, name string, kind Kind
* @return (*Instance, error)
**/
func New(db *jsql.DB, schema, name string, kind Kind) (*Instance, error) {
	columns := []jsql.Column{
		{Name: jsql.CREATED_AT, TypeColumn: jsql.COLUMN, TypeData: jsql.DATETIME, Default: ""},
		{Name: jsql.UPDATED_AT, TypeColumn: jsql.COLUMN, TypeData: jsql.DATETIME, Default: ""},
		{Name: jsql.ID, TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "tag", TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "owner_id", TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "definition", TypeColumn: jsql.COLUMN, TypeData: jsql.BYTES, Default: []byte{}},
	}

	if kind == KindJson {
		columns[5].TypeData = jsql.JSON
	}

	result, err := db.Define(jsql.Def{
		Schema:  schema,
		Name:    name,
		Version: 1,
		Columns: columns,
		PrimaryKeys: []jsql.DefIndex{
			{Name: jsql.ID, Sorted: true},
		},
		Indexes: []jsql.DefIndex{
			{Name: "tag", Sorted: true},
			{Name: "owner_id", Sorted: true},
			{Name: "_idt", Sorted: true},
		},
		IdxField: jsql.IDX,
		IdtField: jsql.IDT,
		IsCore:   true,
		IsDebug:  true,
	})
	if err != nil {
		return nil, err
	}
	result.BeforeInsert(func(tx *jsql.Tx, old, new et.Json) error {
		now := timezone.Now()
		new.Set(jsql.CREATED_AT, now)
		new.Set(jsql.UPDATED_AT, now)
		return nil
	})
	result.BeforeUpdate(func(tx *jsql.Tx, old, new et.Json) error {
		now := timezone.Now()
		new.Set(jsql.UPDATED_AT, now)
		return nil
	})

	err = result.Init()
	if err != nil {
		return nil, err
	}

	return &Instance{model: result}, nil
}

/**
* LoadModel
* @param db *jsql.DB, schema, name string, kind Kind
* @return (*Instance, error)
**/
func Load(db *jsql.DB, schema, name string, kind Kind) (*Instance, error) {
	if instance != nil {
		return instance, nil
	}
	instance, err := New(db, schema, name, kind)
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

	var objData et.Json
	if s.kind == KindJson {
		err := json.Unmarshal(bt, &objData)
		if err != nil {
			return err
		}
	} else {
		objData = et.Json{
			"id":         id,
			"tag":        tag,
			"owner_id":   ownerId,
			"definition": bt,
		}
	}

	_, err := s.model.
		Upsert(objData).
		Where(jsql.Eq("id", id)).
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
		Where(jsql.Eq("id", id)).
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
		Where(jsql.Eq("id", id)).
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
