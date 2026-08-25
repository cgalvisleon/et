package stores

import (
	"fmt"

	"github.com/cgalvisleon/et/dt"
	"github.com/cgalvisleon/et/et"
	. "github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/timezone"
)

type FormState struct {
	TenantId string
	model    *Model
}

/**
* DefineFormState
* @param db *DB, tenantId, schema string
* @return (*FormState, error)
**/
func DefineFormState(db *DB, tenantId, schema string) (*FormState, error) {
	columns := []Column{
		{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "app_id", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "tag", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "title", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
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
			{Name: TENANT_ID, Sorted: true},
			{Name: "app_id", Sorted: true},
			{Name: "tag", Sorted: true},
			{Name: "title", Sorted: true},
		},
		IdxField:    IDX,
		IdtField:    IDT,
		SourceField: SOURCE,
		Details: []DefDetail{
			{
				Name: "owners",
				Keys: map[string]string{
					ID: "state_id",
				},
				Rows: 1,
				Columns: []Column{
					{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
					{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
					{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
					{Name: "owner_id", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
				},
				Indexes: []DefIndex{
					{Name: "owner_id", Sorted: true},
				},
				IdxField: IDX,
				IdtField: IDT,
			},
		},
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

	return &FormState{
		TenantId: tenantId,
		model:    result,
	}, nil
}

/**
* Set
* @param id, tag, ownerId string, obj any
* @return error
**/
func (s *FormState) Set(id, tag, ownerId string, data et.Json) error {
	key := fmt.Sprintf("form_state:%s", id)
	dt.Drop(key)

	data.Set(ID, id)
	data.Set("tag", tag)
	_, err := s.model.
		Upsert(data).
		Where(Eq(ID, id)).
		Exec()
	if err != nil {
		return err
	}

	owner, ok := s.model.Detail("owners")
	if ok {
		_, err := owner.
			Upsert(et.Json{
				"tenant_id": s.TenantId,
				"id":        id,
				"owner_id":  ownerId,
			}).
			Where(Eq(ID, id)).
			And(Eq("owner_id", ownerId)).
			Exec()
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* Get
* @param id string, dest et.Json
* @return (bool, error)
**/
func (s *FormState) Get(id string, dest et.Json) (bool, error) {
	key := fmt.Sprintf("form_state:%s", id)
	item := dt.Get(key)
	if item.Ok {
		result, err := item.Json()
		if err != nil {
			return false, err
		}
		dest = result
		return true, nil
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

	owner, ok := s.model.Detail("owners")
	if ok {
		owners, err := owner.
			Where(Eq(ID, id)).
			All()
		if err != nil {
			return false, err
		}

		result.Set("owners", owners.Result)
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
func (s *FormState) Delete(id string) error {
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
func (s *FormState) Query(query et.Json) (et.Items, error) {
	return s.model.Query(query)
}
