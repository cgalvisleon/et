package jsql

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
)

type State struct {
	model *Model
}

/**
* DefineState
* @param db *DB, schema, name string, kind Kind
* @return (*State, error)
**/
func DefineState(db *DB, schema, name string, kind Kind) (*State, error) {
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

	return &State{model: result}, nil
}

/**
* Set
* @param id, tag, ownerId string, obj any
* @return error
**/
func (s *State) Set(id, tag, ownerId string, state et.Json) error {
	state.Set(ID, id)
	state.Set("tag", tag)
	state.Set("owner_id", ownerId)
	_, err := s.model.
		Upsert(state).
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
func (s *State) Get(id string, dest any) (bool, error) {
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
func (s *State) Delete(id string) error {
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
func (s *State) Query(query et.Json) (et.Items, error) {
	return s.model.Query(query)
}
