package jsql

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
)

func DefineSeries(db *DB, schema string) error {
	if db.series != nil {
		return nil
	}

	var err error
	db.series, err = db.Define(Def{
		Schema:  schema,
		Name:    "series",
		Version: 1,
		Columns: []Column{
			{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
			{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
			{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "tag", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "format", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
			{Name: "value", TypeColumn: COLUMN, TypeData: INT, Default: ""},
		},
		PrimaryKeys: []DefIndex{
			{Name: TENANT_ID, Sorted: true},
			{Name: "tag", Sorted: true},
		},
		IdxField: IDX,
		IsCore:   true,
		IsDebug:  true,
	})
	if err != nil {
		return err
	}

	db.series.
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

	return db.series.Init()
}

/**
* SetSeries
* @param string tag, format string, value int
* @return error
**/
func (s *DB) SetSeries(tag string, format string, value int) error {
	if format == "" {
		format = "%08d"
	}
	_, err := s.series.
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
func (s *DB) GetSeries(tag string) (et.Item, error) {
	result, err := s.series.
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
func (s *DB) DeleteSeries(tag string) error {
	_, err := s.series.
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
func (s *DB) GenSerie(tag string) (string, error) {
	item, err := s.series.
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
func (s *DB) GenValue(tag string) (int, error) {
	item, err := s.series.
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
