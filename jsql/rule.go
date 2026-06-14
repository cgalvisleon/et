package jsql

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/timezone"
)

type Rule struct {
	model *Model
}

/**
* defineRule: Defines the rule table.
* @param db *DB, schema string
* @return (*Rule, error)
**/
func defineRule(db *DB) error {
	if db.rules != nil {
		return nil
	}

	model, err := db.Define(Def{
		Schema:  "core",
		Name:    "rules",
		Version: 1,
		Columns: []Column{
			{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
			{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
			{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "definition", TypeColumn: COLUMN, TypeData: BYTES, Default: []byte{}},
		},
		PrimaryKeys: []DefIndex{
			{Name: ID, Sorted: true},
		},
		IdxField: IDX,
		IsCore:   true,
	})
	if err != nil {
		return err
	}

	model.
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

	err = model.Init()
	if err != nil {
		return err
	}

	db.rules = &Rule{model: model}
	return nil
}

/**
* SetModule: Sets the module data for the given module.
* @param module string, source any
* @return error
**/
func (s *Rule) SetModule(module string, source any) error {
	bt, ok := source.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(source)
		if err != nil {
			return err
		}
	}

	_, err := s.model.
		Upsert(et.Json{
			"id":         module,
			"definition": bt,
		}).
		Where(Eq("id", module)).
		Exec()
	if err != nil {
		return err
	}

	return nil
}

/**
* GetModule: Gets the module data for the given module.
* @param module string, source any
* @return (bool, error)
**/
func (s *Rule) GetModule(module string, source any) (bool, error) {
	item, err := s.model.
		Where(Eq("id", module)).
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
	err = json.Unmarshal(bt, &source)
	if err != nil {
		return false, err
	}

	return true, nil
}

/**
* DeleteModule: Deletes the module data for the given module.
* @param module string
* @return error
**/
func (s *Rule) DeleteModule(module string) error {
	_, err := s.model.
		Delete().
		Where(Eq("id", module)).
		Exec()
	if err != nil {
		return err
	}
	return nil
}

func (s *Rule) Load(tag string) (*jrex.Jrex, error) {
	return nil, nil
}

func (s *Rule) Save(jrex *jrex.Jrex, userId string) error {
	return nil
}

func (s *Rule) GetCode(module string) (string, error) {
	return "", nil
}

func (s *Rule) SetCode(module string, code string) error {
	return nil
}
