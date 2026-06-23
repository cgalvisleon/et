package stores

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/dt"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	. "github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type Authorization struct {
	TenantId string
	model    *Model
}

var (
	ErrorSetAuthor = errors.New(msg.MSG_RECORD_NOT_FOUND)
)

/**
* DefineAuthorization
* @param db *DB, tenantId, schema string
* @return (*Authorization, error)
**/
func DefineAuthorization(db *DB, tenantId, schema string) (*Authorization, error) {
	columns := []Column{
		{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "profile_id", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "method", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
		{Name: "path", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
		{Name: SOURCE, TypeColumn: COLUMN, TypeData: JSON, Default: et.Json{}},
	}

	def := Def{
		Schema:  schema,
		Name:    "authorizations",
		Version: 1,
		Columns: columns,
		PrimaryKeys: []DefIndex{
			{Name: TENANT_ID, Sorted: true},
			{Name: "profile_id", Sorted: true},
			{Name: "method", Sorted: true},
			{Name: "path", Sorted: true},
		},
		Unique: []DefIndex{
			{Name: ID, Sorted: true},
		},
		IdxField:    IDX,
		IdtField:    IDT,
		SourceField: SOURCE,
		IsCore:      true,
	}

	result, err := db.Define(def)
	if err != nil {
		return nil, err
	}
	result.BeforeInsert(func(tx *Tx, old, new et.Json) error {
		tenantId := new.Str(TENANT_ID)
		profileId := new.Str("profile_id")
		method := new.Str("method")
		path := new.Str("path")
		exists, err := result.
			Where(Eq("tenant_id", tenantId)).
			And(Eq("profile_id", profileId)).
			And(Eq("method", method)).
			And(Eq("path", path)).
			ExistsTx(tx)
		if err != nil {
			return err
		}

		if exists {
			return errors.New(MSG_RECORD_EXISTS)
		}

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

	return &Authorization{
		TenantId: tenantId,
		model:    result,
	}, nil
}

/**
* SetAuthor
* @param profileId, method, path string
* @return error
**/
func (s *Authorization) SetAuthor(profileId, method, path string) error {
	if !utility.ValidStr(method, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "method")
	}
	if !utility.ValidStr(path, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "path")
	}

	_, err := s.model.
		Insert(et.Json{
			"tenant_id":  s.TenantId,
			"profile_id": profileId,
			"method":     method,
			"path":       path,
		}).
		One()
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s:%s:%s", s.TenantId, profileId, method, path)
	dt.Drop(key)

	return nil
}

/**
* SetPath
* @params method, path string
* @return error
**/
func (s *Authorization) SetPath(method, path string) error {
	err := s.SetAuthor("", method, path)
	if err != nil && !errors.Is(err, ErrorSetAuthor) {
		return err
	}

	return nil
}

/**
* Author
* @param profileId, method, path string
* @return et.Item, error
**/
func (s *Authorization) Author(profileId, method, path string) (bool, error) {
	key := fmt.Sprintf("%s:%s:%s:%s", s.TenantId, profileId, method, path)
	item := dt.Get(key)
	if item.Ok {
		b, ok := item.Bool()
		if ok {
			return b, nil
		}
	}

	result, err := s.model.
		Where(Eq("tenant_id", s.TenantId)).
		And(Eq("profile_id", profileId)).
		And(Eq("method", method)).
		And(Eq("path", path)).
		Exists()
	if err != nil {
		return false, err
	}

	dt.Up(key, result)
	return result, nil
}

/**
* RemoveAuthor
* @param tenantId, profileId, method, path string
* @return error
**/
func (s *Authorization) RemoveAuthor(profileId, method, path string) error {
	key := fmt.Sprintf("%s:%s:%s:%s", s.TenantId, profileId, method, path)
	dt.Drop(key)

	_, err := s.model.
		Delete().
		Where(Eq("tenant_id", s.TenantId)).
		And(Eq("profile_id", profileId)).
		And(Eq("method", method)).
		And(Eq("path", path)).
		Exec()
	if err != nil {
		return err
	}

	event.Publish(EVENT_DEL_AUTHORIZATION, et.Json{
		"tenant_id":  s.TenantId,
		"profile_id": profileId,
		"method":     method,
		"path":       path,
	})
	return nil
}

/**
* Query
* @param query et.Json
* @return (et.Items, error)
**/
func (s *Authorization) Query(query et.Json) (et.Items, error) {
	return s.model.Query(query)
}
