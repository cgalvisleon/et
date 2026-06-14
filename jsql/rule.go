package jsql

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/timezone"
)

/**
* defineRule: Defines the rule table.
* @param db *DB, schema string
* @return (*Rule, error)
**/
func defineRule(db *DB) error {
	if db.rules != nil {
		return nil
	}

	var err error
	db.rules, err = db.Define(Def{
		Schema:  "core",
		Name:    "rules",
		Version: 1,
		Columns: []Column{
			{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
			{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
			{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "model_id", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "kind", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "definition", TypeColumn: COLUMN, TypeData: BYTES, Default: []byte{}},
		},
		PrimaryKeys: []DefIndex{
			{Name: ID, Sorted: true},
		},
		Indexes: []DefIndex{
			{Name: "model_id", Sorted: true},
			{Name: "kind", Sorted: true},
		},
		IdxField: IDX,
		IsCore:   true,
	})
	if err != nil {
		return err
	}

	db.rules.
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

	return db.rules.Init()
}

type Rule struct {
	modelId string
	rules   *Model
}

/**
* newRule: Constructs a new Rule with initialized fields.
* @param model *Model
* @return *Rule
**/
func newRule(model *Model) *Rule {
	return &Rule{
		modelId: model.Key(),
		rules:   model.db.rules,
	}
}

/**
* setCatalog: Sets the catalog
* @param id, kind string, obj any
* @return string
**/
func (s *Rule) setCatalog(id, kind string, obj any) error {
	bt, ok := obj.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(obj)
		if err != nil {
			return err
		}
	}

	_, err := s.rules.
		Upsert(et.Json{
			"model_id":   s.modelId,
			"kind":       kind,
			"definition": bt,
			"id":         id,
		}).
		Where(Eq("id", id)).
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
func (s *Rule) getCatalog(name, kind string, des any) (bool, error) {
	item, err := s.rules.
		Where(Eq("model_id", s.modelId)).
		And(Eq("kind", kind)).
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

	err = json.Unmarshal(bt, &des)
	if err != nil {
		return false, err
	}

	return true, nil
}

/**
* getModule: Gets the module
* @params module string
* @return *Module, error
**/
func (s *Rule) getModule(module string) (*jrex.Module, error) {
	var result *jrex.Module
	exists, err := s.getCatalog(module, "module", result)
	if err != nil {
		return nil, err
	}

	if exists {
		return result, nil
	}

	result = jrex.NewModule(module)
	err = s.setCatalog(result.ID, "module", result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

/**
* loadModule: Loads the module
* @param module string
* @return *jrex.Module, error
**/
func (s *Rule) Load(tag string) (*jrex.Jrex, error) {
	var result *jrex.Jrex
	exists, err := s.getCatalog(tag, "jrex", result)
	if err != nil {
		return nil, err
	}

	if exists {
		result.Up(s)
		return result, nil
	}

	module, err := s.getModule("index")
	if err != nil {
		return nil, err
	}

	result, err = jrex.NewJrex(tag)
	if err != nil {
		return nil, err
	}
	result.AddModule(module)
	result.Up(s)

	err = s.setCatalog(result.ID, "jrex", result)
	if err != nil {
		return nil, err
	}

	return result, nil
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
