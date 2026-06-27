package stores

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	. "github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/timezone"
)

/**
* Dbs
* @param db *DB,
* @return *Dbs
**/
type Dbs struct {
	model *Model
	dbs   map[string]*DB
}

/**
* DefineDbs
* @param db *DB, schema string
* @return *Dbs, error
**/
func DefineDbs(db *DB, schema string) (*Dbs, error) {
	columns := []Column{
		{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "name", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "host", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
		{Name: "params", TypeColumn: COLUMN, TypeData: JSON, Default: ""},
	}

	def := Def{
		Schema:  schema,
		Name:    "dbs",
		Version: 1,
		Columns: columns,
		PrimaryKeys: []DefIndex{
			{Name: ID, Sorted: true},
		},
		Indexes: []DefIndex{
			{Name: TENANT_ID, Sorted: true},
			{Name: "name", Sorted: true},
			{Name: "host", Sorted: true},
		},
		IdxField: IDX,
	}

	result, err := db.Define(def)
	if err != nil {
		return nil, err
	}
	result.BeforeInsert(func(tx *Tx, old, new et.Json) error {
		tenantId := new.Str(TENANT_ID)
		name := new.Str("name")
		host := new.Str("host")
		exists, err := result.
			Where(Eq("tenant_id", tenantId)).
			And(Eq("name", name)).
			And(Eq("host", host)).
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

	return &Dbs{
		model: result,
		dbs:   make(map[string]*DB),
	}, nil
}
