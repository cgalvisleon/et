package ia

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/timezone"
)

/**
* Store: persistence contract used by the Manager to load and unload
* KnowledgeBase instances outside of memory.
* @param collection, id, ownerId string, obj any
* @return error
**/
type Store interface {
	Set(collection, id, ownerId string, obj any) error
	Get(collection, id string, dest any) (bool, error)
	Delete(collection, id string) error
	Query(query et.Json) (et.Items, error)
}

const collectionKnowledgeBases = "knowledge_bases"

// Storage is a jsql-backed reference implementation of Store, so evicted knowledge
// bases can be persisted to a real database and reloaded later. It follows the same
// "definition BYTES column" shape as jwf.Storage.
// @param TenantId string
type Storage struct {
	TenantId string
	db       *jsql.DB
	models   map[string]*jsql.Model
}

/**
* storeDefine: the shared jsql.Def used for every collection this package persists.
* @param tenantId, schema, name string
* @return jsql.Def
**/
func storeDefine(tenantId, schema, name string) jsql.Def {
	columns := []jsql.Column{
		{Name: jsql.CREATED_AT, TypeColumn: jsql.COLUMN, TypeData: jsql.DATETIME, Default: ""},
		{Name: jsql.UPDATED_AT, TypeColumn: jsql.COLUMN, TypeData: jsql.DATETIME, Default: ""},
		{Name: jsql.TENANT_ID, TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: jsql.ID, TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "owner_id", TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "definition", TypeColumn: jsql.COLUMN, TypeData: jsql.BYTES, Default: []byte("")},
	}

	return jsql.Def{
		TenantId: tenantId,
		Schema:   schema,
		Name:     name,
		Version:  1,
		Columns:  columns,
		PrimaryKeys: []jsql.DefIndex{
			{Name: jsql.ID, Sorted: true},
		},
		Indexes: []jsql.DefIndex{
			{Name: jsql.TENANT_ID, Sorted: true},
			{Name: "owner_id", Sorted: true},
		},
		IdxField: jsql.IDX,
	}
}

/**
* DefineStore: creates (or connects to) the "knowledge_bases" table under schema and
* returns a Storage backed by it.
* @param db *jsql.DB, schema string
* @return *Storage, error
**/
func DefineStore(db *jsql.DB, schema string) (*Storage, error) {
	def := storeDefine(db.TenantId, schema, collectionKnowledgeBases)
	knowledgeBases, err := db.Define(def)
	if err != nil {
		return nil, err
	}
	if err := knowledgeBases.Init(); err != nil {
		return nil, err
	}

	return &Storage{
		TenantId: db.TenantId,
		db:       db,
		models:   map[string]*jsql.Model{collectionKnowledgeBases: knowledgeBases},
	}, nil
}

/**
* Set: upserts obj as the "definition" of id inside collection.
* @param collection, id, ownerId string, obj any
* @return error
**/
func (s *Storage) Set(collection, id, ownerId string, obj any) error {
	model, ok := s.models[collection]
	if !ok {
		return fmt.Errorf(MSG_MODEL_NOT_FOUND, collection)
	}

	bt, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	now := timezone.Now()
	_, err = model.
		Upsert(et.Json{
			"tenant_id":  s.TenantId,
			"id":         id,
			"owner_id":   ownerId,
			"definition": bt,
		}).
		BeforeInsert(func(tx *jsql.Tx, old, new et.Json) error {
			new.Set(jsql.CREATED_AT, now)
			new.Set(jsql.UPDATED_AT, now)
			return nil
		}).
		BeforeUpdate(func(tx *jsql.Tx, old, new et.Json) error {
			new.Set(jsql.UPDATED_AT, now)
			return nil
		}).
		Exec()

	return err
}

/**
* Get: loads the "definition" of id inside collection into dest.
* @param collection, id string, dest any
* @return bool, error
**/
func (s *Storage) Get(collection, id string, dest any) (bool, error) {
	model, ok := s.models[collection]
	if !ok {
		return false, fmt.Errorf(MSG_MODEL_NOT_FOUND, collection)
	}

	item, err := model.Where(jsql.Eq(jsql.ID, id)).One()
	if err != nil {
		return false, err
	}
	if !item.Ok {
		return false, nil
	}

	bt, err := item.Byte("definition")
	if err != nil {
		return false, err
	}

	return true, json.Unmarshal(bt, dest)
}

/**
* Delete: removes id from collection.
* @param collection, id string
* @return error
**/
func (s *Storage) Delete(collection, id string) error {
	model, ok := s.models[collection]
	if !ok {
		return fmt.Errorf(MSG_MODEL_NOT_FOUND, collection)
	}

	_, err := model.Delete().Where(jsql.Eq(jsql.ID, id)).Exec()

	return err
}

/**
* Query: runs query against the "knowledge_bases" collection.
* @param query et.Json
* @return et.Items, error
**/
func (s *Storage) Query(query et.Json) (et.Items, error) {
	model, ok := s.models[collectionKnowledgeBases]
	if !ok {
		return et.Items{}, fmt.Errorf(MSG_MODEL_NOT_FOUND, collectionKnowledgeBases)
	}

	return model.Query(query)
}
