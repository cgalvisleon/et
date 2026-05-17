package jsql

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

/**
* defineCatalog: Defines the catalog table.
* @param db *DB
* @return error
**/
func defineCatalog(db *DB) error {
	if db.catalog != nil {
		return nil
	}

	var err error
	db.catalog, err = db.Define(Def{
		Schema:  "core",
		Name:    "catalog",
		Version: 1,
		PrimaryKeys: []DefIndex{
			{Name: "name", TypeData: TEXT, Default: ""},
		},
		IdxField: IDX,
		Indexes: []DefIndex{
			{Name: "kind", TypeData: KEY, Default: ""},
			{Name: "version", TypeData: INT, Default: 0},
		},
		Columns: []Column{
			{Name: "definition", TypeColumn: COLUMN, TypeData: BYTES, Default: []byte{}},
		},
		IsCore:  true,
		IsDebug: true,
	})
	if err != nil {
		return err
	}
	err = db.catalog.Init()
	if err != nil {
		return err
	}

	return nil
}

/**
* setCatalog: Sets the catalog data for the given name.
* @param name, kind string, version int, obj any
* @return error
**/
func (db *DB) setCatalog(name, kind string, version int, obj any) error {
	if db.catalog == nil {
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

	_, err := db.catalog.
		Upsert(et.Json{
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
func (db *DB) getCatalog(name, kind string, des any) error {
	item, err := db.catalog.
		Where(Eq("name", name)).
		And(Eq("kind", kind)).
		One()
	if err != nil {
		return err
	}

	if !item.Ok {
		return fmt.Errorf(msg.MSG_CATALOG_NOT_FOUND, name)
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
