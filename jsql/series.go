package jsql

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
)

type Series struct {
	model  *Model
	schema string
}

/**
* defineSeries
* @param schema string
* @return error
**/
func defineSeries(db *DB, schema string) (*Series, error) {
	columns := []Column{
		{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: et.DATETIME, Default: ""},
		{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: et.DATETIME, Default: ""},
		{Name: "tag", TypeColumn: COLUMN, TypeData: et.KEY, Default: ""},
		{Name: "format", TypeColumn: COLUMN, TypeData: et.TEXT, Default: ""},
		{Name: "value", TypeColumn: COLUMN, TypeData: et.INT, Default: ""},
	}

	def := Define{
		Schema:  schema,
		Name:    "series",
		Version: 1,
		Columns: columns,
		PrimaryKeys: []DefIndex{
			{Name: "tag", Sorted: true},
		},
		IdxField: IDX,
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
		model:  model,
		schema: schema,
	}, nil
}

/**
* setSeries
* @param string tag, format string, value int
* @return error
**/
func (s *Series) setSeries(tag string, format string, value int) error {
	if format == "" {
		format = "%08d"
	}
	_, err := s.model.
		Upsert(et.Json{
			"tag":    tag,
			"format": format,
			"value":  value,
		}).
		Where(Eq("tag", tag)).
		Exec()
	return err
}

/**
* getSeries
* @param string tag, ownerId string
* @return (et.Item, error)
**/
func (s *Series) getSeries(tag string) (et.Item, error) {
	result, err := s.model.
		Where(Eq("tag", tag)).
		One()
	if err != nil {
		return et.Item{}, err
	}
	return result, nil
}

/**
* deleteSeries
* @param string tag, ownerId string
* @return error
**/
func (s *Series) deleteSeries(tag string) error {
	_, err := s.model.
		Delete().
		Where(Eq("tag", tag)).
		Exec()
	if err != nil {
		return err
	}
	return nil
}

/**
* genSerie
* @param string tag
* @return (string, error)
**/
func (s *Series) genSerie(tag string) (string, error) {
	item, err := s.model.
		Upsert(et.Json{}).
		BeforeInsert(func(tx *Tx, old, new et.Json) error {
			new["tag"] = tag
			new["format"] = "%08d"
			new["value"] = 1
			return nil
		}).
		BeforeUpdate(func(tx *Tx, old, new et.Json) error {
			value := old.Int("value")
			new["value"] = value + 1
			return nil
		}).
		Where(Eq("tag", tag)).
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
* genValue
* @param string tag
* @return (int, error)
**/
func (s *Series) genValue(tag string) (int, error) {
	item, err := s.model.
		Update(et.Json{}).
		BeforeUpdate(func(tx *Tx, old, new et.Json) error {
			new["value"] = old.Int("value") + 1
			return nil
		}).
		Where(Eq("tag", tag)).
		One()
	if err != nil {
		return 0, err
	}

	return item.Int("value"), nil
}
