package stores

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/dt"
	"github.com/cgalvisleon/et/et"
	. "github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/timezone"
)

type Kind string

type Instance struct {
	TenantId string
	model    *Model
}

/**
* DefineInstance
* @param db *DB, tenantId, schema string
* @return (*Instance, error)
**/
func DefineInstance(db *DB, tenantId, schema string) (*Instance, error) {
	columns := []Column{
		{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "tag", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "title", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
		{Name: "owner_id", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "definition", TypeColumn: COLUMN, TypeData: BYTES, Default: []byte("")},
	}

	def := Def{
		Schema:  schema,
		Name:    "instances",
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

	return &Instance{
		TenantId: tenantId,
		model:    result,
	}, nil
}

/**
* Set
* @param tag, id, ownerId string, obj any
* @return error
**/
func (s *Instance) Set(tag, id, ownerId string, obj any) error {
	bt, ok := obj.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(obj)
		if err != nil {
			return err
		}
	}

	_, err := s.model.
		Upsert(et.Json{
			"tenant_id":  s.TenantId,
			"id":         id,
			"tag":        tag,
			"owner_id":   ownerId,
			"definition": bt,
		}).
		BeforeInsert(func(tx *Tx, old, new et.Json) error {
			now := timezone.Now()
			new.Set(CREATED_AT, now)
			new.Set(UPDATED_AT, now)
			return nil
		}).
		BeforeUpdate(func(tx *Tx, old, new et.Json) error {
			now := timezone.Now()
			new.Set(UPDATED_AT, now)
			return nil
		}).
		Where(Eq(ID, id)).
		Exec()
	if err != nil {
		return err
	}

	key := fmt.Sprintf("instance:%s", id)
	dt.Drop(key)

	return nil
}

/**
* Get
* @param id string, dest any
* @return (bool, error)
**/
func (s *Instance) Get(id string, dest any) (bool, error) {
	key := fmt.Sprintf("instance:%s", id)
	result := dt.Get(key)
	var bt []byte
	if result.Ok {
		item, ok := result.Item()
		if !ok {
			return false, errors.New(MSG_RECORD_IS_NOT_ITEM)
		}

		var err error
		bt, err = item.Byte("definition")
		if err != nil {
			return false, err
		}
	} else {
		item, err := s.model.
			Where(Eq(ID, id)).
			One()
		if err != nil {
			return false, err
		}

		if !item.Ok {
			return false, errors.New(MSG_RECORD_NOT_FOUND)
		}

		bt, err = item.Byte("definition")
		if err != nil {
			return false, err
		}
	}

	err := json.Unmarshal(bt, &dest)
	if err != nil {
		return false, err
	}

	dt.Up(key, bt)
	return true, nil
}

/**
* Delete
* @param id string
* @return error
**/
func (s *Instance) Delete(id string) error {
	key := fmt.Sprintf("instance:%s", id)
	dt.Drop(key)

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
