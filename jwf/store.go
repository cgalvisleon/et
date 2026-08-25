package jwf

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/timezone"
)

const (
	storeWorkflows = "workflows"
	storeFlows     = "flows"
	storeInstances = "instances"
	storeSteps     = "steps"
)

type Store interface {
	Set(collection, id, ownerId string, obj any) error
	SetInstance(id, workflowId, projectId, flowId, flowTag, code, title string, status Status, obj any) error
	Get(collection, id string, dest any) (bool, error)
	Delete(collection, id string) error
	Query(collection string, query et.Json) (et.Items, error)
	// Series
	SetSeries(tag string, format string, value int) error
	GetSeries(tag string) (et.Item, error)
	DeleteSeries(tag string) error
	GenSerie(tag string) (string, error)
	GenValue(tag string) (int, error)
}

type Storage struct {
	db     *jsql.DB
	series *jsql.Series
	models map[string]*jsql.Model
}

/**
* storeDefine
* @param schema, name string
* @return jsql.Def
**/
func storeDefine(schema, name string) jsql.Def {
	columns := []jsql.Column{
		{Name: jsql.CREATED_AT, TypeColumn: jsql.COLUMN, TypeData: jsql.DATETIME, Default: ""},
		{Name: jsql.UPDATED_AT, TypeColumn: jsql.COLUMN, TypeData: jsql.DATETIME, Default: ""},
		{Name: jsql.ID, TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "owner_id", TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "definition", TypeColumn: jsql.COLUMN, TypeData: jsql.BYTES, Default: []byte("")},
	}

	def := jsql.Def{
		Schema:  schema,
		Name:    name,
		Version: 1,
		Columns: columns,
		PrimaryKeys: []jsql.DefIndex{
			{Name: jsql.ID, Sorted: true},
		},
		Indexes: []jsql.DefIndex{
			{Name: "owner_id", Sorted: true},
		},
		IdxField: jsql.IDX,
	}

	return def
}

/**
* storeInstance
* @param schema, name string
* @return jsql.Def
**/
func storeInstance(schema, name string) jsql.Def {
	columns := []jsql.Column{
		{Name: jsql.CREATED_AT, TypeColumn: jsql.COLUMN, TypeData: jsql.DATETIME, Default: ""},
		{Name: jsql.UPDATED_AT, TypeColumn: jsql.COLUMN, TypeData: jsql.DATETIME, Default: ""},
		{Name: jsql.ID, TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "workflow_id", TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "project_id", TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "flow_id", TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "flow_tag", TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "code", TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "title", TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "status", TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "definition", TypeColumn: jsql.COLUMN, TypeData: jsql.BYTES, Default: []byte("")},
	}

	def := jsql.Def{
		Schema:  schema,
		Name:    name,
		Version: 1,
		Columns: columns,
		PrimaryKeys: []jsql.DefIndex{
			{Name: jsql.ID, Sorted: true},
		},
		Indexes: []jsql.DefIndex{
			{Name: "workflow_id", Sorted: true},
			{Name: "project_id", Sorted: true},
			{Name: "flow_id", Sorted: true},
			{Name: "flow_tag", Sorted: true},
			{Name: "code", Sorted: true},
			{Name: "title", Sorted: true},
			{Name: "status", Sorted: true},
		},
		IdxField: jsql.IDX,
	}

	return def
}

/**
* DefineStore
* @param db *jsql.DB, schema string
* @return *Storage, error
**/
func DefineStore(db *jsql.DB, schema string) (*Storage, error) {
	def := storeDefine(schema, storeWorkflows)
	workflows, err := db.Define(def)
	if err != nil {
		return nil, err
	}
	err = workflows.Init()
	if err != nil {
		return nil, err
	}

	def = storeDefine(schema, storeFlows)
	flows, err := db.Define(def)
	if err != nil {
		return nil, err
	}
	err = flows.Init()
	if err != nil {
		return nil, err
	}

	def = storeDefine(schema, storeSteps)
	steps, err := db.Define(def)
	if err != nil {
		return nil, err
	}
	err = steps.Init()
	if err != nil {
		return nil, err
	}

	result := &Storage{
		db:     db,
		models: make(map[string]*jsql.Model),
	}
	result.series, err = jsql.DefineSeries(db, schema)
	if err != nil {
		return nil, err
	}
	result.models[storeWorkflows] = workflows
	result.models[storeFlows] = flows
	result.models[storeSteps] = steps

	return result, nil
}

/**
* DefineInstances
* @param db *jsql.DB, dbId string
* @return error
**/
func (s *Storage) DefineInstances(db *jsql.DB) error {
	def := storeInstance(storeWorkflows, storeInstances)
	result, err := db.Define(def)
	if err != nil {
		return err
	}
	err = result.Init()
	if err != nil {
		return err
	}
	s.models[storeInstances] = result
	return nil
}

/**
* Set
* @param collection, id, ownerId string, obj any
* @return error
**/
func (s *Storage) Set(collection, id, ownerId string, obj any) error {
	bt, ok := obj.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(obj)
		if err != nil {
			return err
		}
	}

	model, ok := s.models[collection]
	if !ok {
		return fmt.Errorf(MSG_MODEL_NOT_FOUND, collection)
	}

	now := timezone.Now()
	_, err := model.
		Upsert(et.Json{
			"kind":       collection,
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
		Where(jsql.Eq(jsql.ID, id)).
		Exec()
	if err != nil {
		return err
	}
	return nil
}

/**
* SetInstance
* @param id, workflowId, projectId, flowId, flowTag, code, title string, status Status, obj any
* @return error
**/
func (s *Storage) SetInstance(id, workflowId, projectId, flowId, flowTag, code, title string, status Status, obj any) error {
	bt, ok := obj.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(obj)
		if err != nil {
			return err
		}
	}

	model, ok := s.models[storeInstances]
	if !ok {
		return fmt.Errorf(MSG_MODEL_NOT_FOUND, storeInstances)
	}

	now := timezone.Now()
	_, err := model.
		Upsert(et.Json{
			"workflow_id": workflowId,
			"project_id":  projectId,
			"flow_id":     flowId,
			"flow_tag":    flowTag,
			"code":        code,
			"title":       title,
			"status":      status.Str(),
			"id":          id,
			"definition":  bt,
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
		Where(jsql.Eq(jsql.ID, id)).
		Exec()
	if err != nil {
		return err
	}
	return nil
}

/**
* Get
* @param collection, id string, dest any
* @return (bool, error)
**/
func (s *Storage) Get(collection, id string, dest any) (bool, error) {
	model, ok := s.models[collection]
	if !ok {
		return false, fmt.Errorf(MSG_MODEL_NOT_FOUND, collection)
	}

	item, err := model.
		Where(jsql.Eq(jsql.ID, id)).
		One()
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

	err = json.Unmarshal(bt, &dest)
	if err != nil {
		return false, err
	}

	return true, nil
}

/**
* Delete
* @param collection, id string
* @return error
**/
func (s *Storage) Delete(collection, id string) error {
	model, ok := s.models[collection]
	if !ok {
		return fmt.Errorf(MSG_MODEL_NOT_FOUND, collection)
	}

	_, err := model.
		Delete().
		Where(jsql.Eq(jsql.ID, id)).
		Exec()
	if err != nil {
		return err
	}
	return nil
}

/**
* Query
* @param collection string, query et.Json
* @return et.Items, error
**/
func (s *Storage) Query(collection string, query et.Json) (et.Items, error) {
	model, ok := s.models[collection]
	if !ok {
		return et.Items{}, fmt.Errorf(MSG_MODEL_NOT_FOUND, collection)
	}

	return model.Query(query)
}

/**
* SetSeries
* @param tag string, format string, value int
* @return error
**/
func (s *Storage) SetSeries(tag string, format string, value int) error {
	return s.series.SetSeries(tag, format, value)
}

/**
* GetSeries
* @param tag string
* @return et.Item, error
**/
func (s *Storage) GetSeries(tag string) (et.Item, error) {
	return s.series.GetSeries(tag)
}

/**
* DeleteSeries
* @param tag string
* @return error
**/
func (s *Storage) DeleteSeries(tag string) error {
	return s.series.DeleteSeries(tag)
}

/**
* GenSerie
* @param tag string
* @return string, error
**/
func (s *Storage) GenSerie(tag string) (string, error) {
	return s.series.GenSerie(tag)
}

/**
* GenValue
* @param tag string
* @return int, error
**/
func (s *Storage) GenValue(tag string) (int, error) {
	return s.series.GenValue(tag)
}
