package stores

import (
	"github.com/cgalvisleon/et/dt"
	"github.com/cgalvisleon/et/et"
	. "github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/timezone"
)

type State struct {
	model *Model
}

/**
* DefineState
* @param db *DB, schema, name string
* @return (*State, error)
**/
func DefineState(db *DB, schema string) (*State, error) {
	columns := []Column{
		{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "tag", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "title", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
		{Name: "owner_id", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: SOURCE, TypeColumn: COLUMN, TypeData: JSON, Default: et.Json{}},
	}

	def := Def{
		Schema:  schema,
		Name:    "states",
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
		IdxField:    IDX,
		IdtField:    IDT,
		SourceField: SOURCE,
		IsCore:      true,
		IsDebug:     true,
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

	dt.Up(id, state)

	return nil
}

/**
* Get
* @param id string, dest et.Json
* @return (bool, error)
**/
func (s *State) Get(id string, dest et.Json) (bool, error) {
	item := dt.Get(id)
	if item.Ok {
		var ok bool
		dest, ok = item.Json()
		if ok {
			return true, nil
		}
	}

	result, err := s.model.
		Where(Eq(ID, id)).
		One()
	if err != nil {
		return false, err
	}

	if !result.Ok {
		return false, nil
	}

	dest = result.Result
	dt.Up(id, result.Result)

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

	dt.Drop(id)

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
