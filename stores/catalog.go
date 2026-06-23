package stores

import (
	"encoding/json"
	"errors"

	"github.com/cgalvisleon/et/et"
	. "github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/timezone"
)

type Catalog struct {
	TenantId string
	model    *Model
}

/**
* DefineCatalog: Defines the catalog table.
* @param db *DB
* @return error
**/
func DefineCatalog(db *DB, tenantId, schema string) (*Catalog, error) {
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

	return &Catalog{
		TenantId: tenantId,
		model:    result,
	}, nil
}

/**
* Set: Sets the catalog data for the given name.
* @param collection, id, ownerId string, obj any
* @return error
**/
func (s *Catalog) Set(collection, id, ownerId string, obj any) error {
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
* @return error
**/
func (s *Catalog) Get(collection, id string, des any) error {
	item, err := s.model.
		Where(Eq("tenant_id", s.TenantId)).
		And(Eq("kind", collection)).
		And(Eq("id", id)).
		One()
	if err != nil {
		return err
	}

	if !item.Ok {
		return errors.New(MSG_RECORD_NOT_FOUND)
	}

	bt, err := item.Byte("definition")
	if err != nil {
		return err
	}

	err = json.Unmarshal(bt, &des)
	if err != nil {
		return err
	}

	return nil
}

/**
* Delete: Deletes the catalog data for the given name.
* @param collection, id string
* @return error
**/
func (s *Catalog) Delete(collection, id string) error {
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
func (s *Catalog) Query(query et.Json) (et.Items, error) {
	return s.model.Query(query)
}
