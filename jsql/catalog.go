package jsql

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
)

/**
* DefineCatalog: Defines the catalog table.
* @param db *DB
* @return error
**/
func DefineCatalog(db *DB, schema string) error {
	if db.catalog != nil {
		return nil
	}

	var err error
	db.catalog, err = db.Define(Def{
		Schema:  schema,
		Name:    "catalog",
		Version: 1,
		Columns: []Column{
			{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
			{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
			{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "name", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
			{Name: "kind", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "version", TypeColumn: COLUMN, TypeData: INT, Default: 0},
			{Name: "definition", TypeColumn: COLUMN, TypeData: BYTES, Default: []byte{}},
		},
		PrimaryKeys: []DefIndex{
			{Name: TENANT_ID, Sorted: true},
			{Name: "name", Sorted: true},
			{Name: "kind", Sorted: true},
		},
		IdxField: IDX,
		Indexes: []DefIndex{
			{Name: TENANT_ID, Sorted: true},
			{Name: "kind", Sorted: true},
			{Name: "version", Sorted: true},
		},
		IsCore: true,
	})
	if err != nil {
		return err
	}

	db.catalog.
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
		})

	return db.catalog.Init()
}

/**
* setCatalog: Sets the catalog data for the given name.
* @param name, kind string, version int, obj any
* @return error
**/
func (s *DB) setCatalog(name, kind string, version int, obj any) error {
	if s.catalog == nil {
		return nil
	}

	bt, ok := obj.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(obj)
		if err != nil {
			return err
		}
	}

	_, err := s.catalog.
		Upsert(et.Json{
			"tenant_id":  s.TenantId,
			"name":       name,
			"kind":       kind,
			"version":    version,
			"definition": bt,
		}).
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
func (s *DB) getCatalog(name, kind string, des any) error {
	item, err := s.catalog.
		Where(Eq("tenant_id", s.TenantId)).
		Where(Eq("name", name)).
		And(Eq("kind", kind)).
		One()
	if err != nil {
		return err
	}

	if !item.Ok {
		return fmt.Errorf(MSG_CATALOG_NOT_FOUND, name)
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
* deleteCatalog: Deletes the catalog data for the given name.
* @param name, kind string
* @return error
**/
func (s *DB) deleteCatalog(name, kind string) error {
	_, err := s.catalog.
		Delete().
		Where(Eq("tenant_id", s.TenantId)).
		And(Eq("name", name)).
		And(Eq("kind", kind)).
		Exec()
	if err != nil {
		return err
	}
	return nil
}
