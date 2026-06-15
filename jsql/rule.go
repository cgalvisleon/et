package jsql

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/reg"
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
			{Name: "owner_id", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "kind", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "tag", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "definition", TypeColumn: COLUMN, TypeData: BYTES, Default: []byte{}},
		},
		PrimaryKeys: []DefIndex{
			{Name: "owner_id", Sorted: true},
			{Name: "kind", Sorted: true},
			{Name: "tag", Sorted: true},
		},
		Unique: []DefIndex{
			{Name: ID, Sorted: true},
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
			id := new.String(ID)
			id = reg.GetULID(id)
			new.Set(CREATED_AT, now)
			new.Set(UPDATED_AT, now)
			new.Set(ID, id)
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
	tenantId string
	ownerId  string
	rules    *Model
}

/**
* loadRule: Loads a Rule from the database catalog by name.
* @param model *Model
* @return *Rule
**/
func loadRule(db *DB, tenantId, ownerId string) *Rule {
	return &Rule{
		tenantId: tenantId,
		ownerId:  ownerId,
		rules:    db.rules,
	}
}

/**
* setCatalog: Sets the catalog
* @param id, kind, tag string, obj any
* @return string
**/
func (s *Rule) setCatalog(id, kind, tag string, obj any) error {
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
			"tenant_id":  s.tenantId,
			"owner_id":   s.ownerId,
			"kind":       kind,
			"tag":        tag,
			"id":         id,
			"definition": bt,
		}).
		Where(Eq("owner_id", s.ownerId)).
		And(Eq("kind", kind)).
		And(Eq("tag", tag)).
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
func (s *Rule) getCatalog(kind, tag string, des any) (bool, error) {
	item, err := s.rules.
		Where(Eq("owner_id", s.ownerId)).
		And(Eq("kind", kind)).
		And(Eq("tag", tag)).
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
* NewModule: Creates a new module
* @param jrex *jrex.Jrex, path string
* @return *jrex.Module, error
**/
func (s *Rule) NewModule(jrex *jrex.Jrex, path string) (*jrex.Module, error) {
	result := jrex.NewModule(path)
	jrex.AddModule(result)
	code := ""
	err := s.setCatalog(result.ID, "code", result.Path, code)
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
	exists, err := s.getCatalog("jrex", tag, result)
	if err != nil {
		return nil, err
	}

	if exists {
		result.Up(s)
		return result, nil
	}

	result, err = jrex.NewJrex(tag)
	if err != nil {
		return nil, err
	}
	_, err = s.NewModule(result, "index")
	if err != nil {
		return nil, err
	}
	result.Up(s)

	err = s.setCatalog(result.ID, "jrex", result.Tag, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* save: Saves the jrex
* @param jrex *jrex.Jrex, userId string
* @return error
**/
func (s *Rule) Save(jrex *jrex.Jrex, userId string) error {
	err := s.setCatalog(jrex.ID, "jrex", jrex.Tag, jrex)
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
	var result string
	exists, err := s.getCatalog("code", module, &result)
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
func (s *Rule) SetCode(module *jrex.Module, code string) error {
	err := s.setCatalog(module.ID, "code", module.Path, code)
	if err != nil {
		return err
	}
	return nil
}
