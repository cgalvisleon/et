package stores

import (
	"encoding/json"

	"github.com/cgalvisleon/et/dt"
	"github.com/cgalvisleon/et/et"
	. "github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/timezone"
)

type Kind string

const (
	KindJson Kind = "json"
	KindBite Kind = "binary"
)

type Instance struct {
	model *Model
	kind  Kind
}

/**
* defineInstance
* @param db *DB, schema, name string, kind Kind
* @return (*Instance, error)
**/
func defineInstance(db *DB, schema, name string, kind Kind) (*Instance, error) {
	columns := []Column{
		{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "tag", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "title", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
		{Name: "owner_id", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: SOURCE, TypeColumn: COLUMN, TypeData: JSON, Default: et.Json{}},
	}

	if name == "" {
		name = "instances"
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
			{Name: TENANT_ID, Sorted: true},
			{Name: "tag", Sorted: true},
			{Name: "title", Sorted: true},
			{Name: "owner_id", Sorted: true},
		},
		IdxField: IDX,
		IdtField: IDT,
		IsCore:   true,
		IsDebug:  true,
	}

	if kind == KindBite {
		def.Columns[5].TypeData = BYTES
	} else {
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

	return &Instance{model: result, kind: kind}, nil
}

/**
* DefineInstance
* @param db *DB, schema, name string
* @return (*Instance, error)
**/
func DefineInstance(db *DB, schema, name string) (*Instance, error) {
	return defineInstance(db, schema, name, KindJson)
}

/**
* DefineInstanceBite
* @param db *DB, schema, name string
* @return (*Instance, error)
**/
func DefineInstanceBite(db *DB, schema, name string) (*Instance, error) {
	return defineInstance(db, schema, name, KindBite)
}

/**
* LoadInstance
* @param db *DB, schema string
* @return (*Instance, error)
**/
func LoadInstance(db *DB, schema string) (*Instance, error) {
	return DefineInstance(db, schema, "instances")
}

/**
* LoadInstanceBite
* @param db *DB, schema string
* @return (*Instance, error)
**/
func LoadInstanceBite(db *DB, schema string) (*Instance, error) {
	return DefineInstanceBite(db, schema, "instances")
}

/**
* Set
* @param id, tag, tenantId, ownerId string, obj any
* @return error
**/
func (s *Instance) Set(id, tag, tenantId, ownerId string, obj any, userId string) error {
	bt, ok := obj.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(obj)
		if err != nil {
			return err
		}
	}

	var data = et.Json{}
	if s.kind == KindBite {
		data = et.Json{
			SOURCE: bt,
		}
	} else {
		err := json.Unmarshal(bt, &data)
		if err != nil {
			return err
		}
	}

	data.Set(TENANT_ID, tenantId)
	data.Set(ID, id)
	data.Set("tag", tag)
	data.Set("owner_id", ownerId)
	data.Set("user_id", userId)
	_, err := s.model.
		Upsert(data).
		Where(Eq(ID, id)).
		Exec()
	if err != nil {
		return err
	}

	dt.Drop(id)

	return nil
}

/**
* Get
* @param id string, dest any
* @return (bool, error)
**/
func (s *Instance) Get(id string, dest any) (bool, error) {
	var item et.Item
	result := dt.Get(id)
	if result.Ok {
		var ok bool
		item, ok = result.Item()
		if !ok {
			item = et.Item{}
		}
	}

	if !item.Ok {
		var err error
		item, err = s.model.
			Where(Eq(ID, id)).
			One()
		if err != nil {
			return false, err
		}
	}

	if !item.Ok {
		return false, nil
	}

	if s.kind == KindBite {
		bt, err := item.Byte(SOURCE)
		err = json.Unmarshal(bt, dest)
		if err != nil {
			return false, err
		}
	} else {
		bt := []byte(item.Result.ToString())
		err := json.Unmarshal(bt, dest)
		if err != nil {
			return false, err
		}
	}

	dt.Up(id, item)
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

	dt.Drop(id)

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
