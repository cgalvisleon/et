package jsql

import (
	"encoding/json"
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
)

type Store struct {
	TenantId string
	model    *Model
}

/**
* defineCatalog: Defines the catalog table.
* @param db *DB, tenantId, schema string
* @return *Catalog, error
**/
func DefineStore(db *DB, tenantId, schema string) (*Store, error) {
	columns := []Column{
		{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "kind", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "id", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "owner_id", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "definition", TypeColumn: COLUMN, TypeData: BYTES, Default: []byte{}},
	}

	def := Def{
		Schema:  schema,
		Name:    "catalogs",
		Version: 1,
		Columns: columns,
		PrimaryKeys: []DefIndex{
			{Name: TENANT_ID, Sorted: true},
			{Name: "kind", Sorted: true},
			{Name: "id", Sorted: true},
		},
		Indexes: []DefIndex{
			{Name: "owner_id", Sorted: true},
		},
		IdxField: IDX,
		IdtField: IDT,
	}

	model, err := db.Define(def)
	if err != nil {
		return nil, err
	}

	now := timezone.Now()
	model.BeforeInsert(func(tx *Tx, old, new et.Json) error {
		new.Set(CREATED_AT, now)
		new.Set(UPDATED_AT, now)
		return nil
	})
	model.BeforeUpdate(func(tx *Tx, old, new et.Json) error {
		new.Set(UPDATED_AT, now)
		return nil
	})

	err = model.Init()
	if err != nil {
		return nil, err
	}

	return &Store{
		TenantId: tenantId,
		model:    model,
	}, nil
}

/**
* Set: Sets the catalog data for the given name.
* @param collection, id, ownerId string, obj any
* @return error
**/
func (s *Store) Set(collection, id, ownerId string, obj any) error {
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
			"kind":       collection,
			"id":         id,
			"owner_id":   ownerId,
			"definition": bt,
		}).
		Where(Eq("tenant_id", s.TenantId)).
		And(Eq("kind", collection)).
		And(Eq("id", id)).
		Exec()
	if err != nil {
		return err
	}

	return nil
}

/**
* getCatalog: Gets the catalog data for the given name.
* @param name, kind string, des any
* @return bool, error
**/
func (s *Store) Get(collection, id string, des any) (bool, error) {
	item, err := s.model.
		Where(Eq("tenant_id", s.TenantId)).
		And(Eq("kind", collection)).
		And(Eq("id", id)).
		One()
	if err != nil {
		return false, err
	}

	if !item.Ok {
		return false, errors.New(MSG_RECORD_NOT_FOUND)
	}

	bt, err := item.Byte("definition")
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(bt, &des)
	if err != nil {
		return false, err
	}

	return true, nil
}

/**
* Delete: Deletes the catalog data for the given name.
* @param collection, id string
* @return error
**/
func (s *Store) Delete(collection, id string) error {
	_, err := s.model.
		Delete().
		Where(Eq("tenant_id", s.TenantId)).
		And(Eq("kind", collection)).
		And(Eq("id", id)).
		Exec()
	if err != nil {
		return err
	}
	return nil
}

/**
* Query: Queries the catalog data for the given name.
* @param query et.Json
* @return (et.Items, error)
**/
func (s *Store) Query(query et.Json) (et.Items, error) {
	return s.model.Query(query)
}
