package jsql

import (
	"encoding/json"
	"fmt"

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
	jrexId  string
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
* getCatalog: Gets the catalog data for the given id.
* @param id, kind string, des any
* @return error
**/
func (s *Rule) getCatalog(id, kind string, des any) (bool, error) {
	item, err := s.rules.
		Where(Eq("model_id", s.modelId)).
		And(Eq("kind", kind)).
		And(Eq("id", id)).
		Debug().
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

	module := jrex.NewModule("index")
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

	s.jrexId = result.ID
	return result, nil
}

/**
* save: Saves the jrex
* @param jrex *jrex.Jrex, userId string
* @return error
**/
func (s *Rule) Save(jrex *jrex.Jrex, userId string) error {
	err := s.setCatalog(jrex.ID, "jrex", jrex)
	if err != nil {
		return err
	}
	return nil
}

/**
* getCode: Gets the code
* @param module string
* @return string, error
**/
func (s *Rule) GetCode(module string) (string, error) {
	id := fmt.Sprintf("%s:%s", s.jrexId, module)
	var result string
	exists, err := s.getCatalog(id, "code", &result)
	if err != nil {
		return "", err
	}

	if exists {
		return result, nil
	}

	return "", nil
}

/**
* setCode: Sets the code
* @param module string, code string
* @return error
**/
func (s *Rule) SetCode(module string, code string) error {
	id := fmt.Sprintf("%s:%s", s.jrexId, module)
	err := s.setCatalog(id, "code", code)
	if err != nil {
		return err
	}
	return nil
}
