package jsql

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
)

type Series struct {
	TenantId string
	model    *Model
}

/**
* defineSeries
* @param schema string
* @return error
**/
func DefineSeries(db *DB, tenantId, schema string) (*Series, error) {
	columns := []Column{
		{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "tag", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "format", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
		{Name: "value", TypeColumn: COLUMN, TypeData: INT, Default: ""},
	}

	def := Def{
		Schema:  schema,
		Name:    "series",
		Version: 1,
		Columns: columns,
		PrimaryKeys: []DefIndex{
			{Name: TENANT_ID, Sorted: true},
			{Name: "tag", Sorted: true},
		},
		IdxField: IDX,
		IsCore:   true,
	}
	model, err := db.Define(def)
	if err != nil {
		return nil, err
	}

	model.BeforeInsert(func(tx *Tx, old, new et.Json) error {
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
		return nil, err
	}

	return &Series{
		TenantId: tenantId,
		model:    model,
	}, nil
}

/**
* SetSeries
* @param string tag, format string, value int
* @return error
**/
func (s *Series) SetSeries(tag string, format string, value int) error {
	if format == "" {
		format = "%08d"
	}
	_, err := s.model.
		Upsert(
			et.Json{
				"tenant_id": s.TenantId,
				"tag":       tag,
				"format":    format,
				"value":     value,
			}).
		Where(Eq("tenant_id", s.TenantId)).
		And(Eq("tag", tag)).
		Exec()
	return err
}

/**
* GetSeries
* @param string tag, ownerId string
* @return (et.Item, error)
**/
func (s *Series) GetSeries(tag string) (et.Item, error) {
	result, err := s.model.
		Where(Eq("tenant_id", s.TenantId)).
		And(Eq("tag", tag)).
		One()
	if err != nil {
		return et.Item{}, err
	}
	return result, nil
}

/**
* DeleteSeries
* @param string tag, ownerId string
* @return error
**/
func (s *Series) DeleteSeries(tag string) error {
	_, err := s.model.
		Delete().
		Where(Eq("tenant_id", s.TenantId)).
		And(Eq("tag", tag)).
		Exec()
	if err != nil {
		return err
	}
	return nil
}

/**
* GenSerie
* @param string tag
* @return (string, error)
**/
func (s *Series) GenSerie(tag string) (string, error) {
	item, err := s.model.
		Update(et.Json{}).
		BeforeUpdate(func(tx *Tx, old, new et.Json) error {
			new["value"] = old["value"].(int) + 1
			return nil
		}).
		Where(Eq("tenant_id", s.TenantId)).
		And(Eq("tag", tag)).
		One()
	if err != nil {
		return "", err
	}
	format := item.String("format")
	value := item.Int("value")
	result := fmt.Sprintf(format, value)
	return result, nil
}

/**
* GenValue
* @param string tag
* @return (int, error)
**/
func (s *Series) GenValue(tag string) (int, error) {
	item, err := s.model.
		Update(et.Json{}).
		BeforeUpdate(func(tx *Tx, old, new et.Json) error {
			new["value"] = old["value"].(int) + 1
			return nil
		}).
		Where(Eq("tenant_id", s.TenantId)).
		And(Eq("tag", tag)).
		One()
	if err != nil {
		return 0, err
	}

	return item.Int("value"), nil
}
