package jsql

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
)

func defineSeries(db *DB) error {
	if db.series != nil {
		return nil
	}

	var err error
	db.series, err = db.Define(Def{
		Schema:  "core",
		Name:    "series",
		Version: 1,
		PrimaryKeys: []DefIndex{
			{Name: "tag", TypeData: KEY, Default: ""},
			{Name: "owner_id", TypeData: KEY, Default: ""},
		},
		Columns: []Column{
			{Name: "format", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
			{Name: "value", TypeColumn: COLUMN, TypeData: INT, Default: ""},
		},
		IdxField: IDX,
		IsCore:   true,
		IsDebug:  true,
	})
	if err != nil {
		return err
	}
	err = db.series.Init()
	if err != nil {
		return err
	}

	return nil
}

/**
* SetSeries
* @param string tag, ownerId string, format string, value int
* @return error
**/
func (db *DB) SetSeries(tag, ownerId string, format string, value int) error {
	if format == "" {
		format = "%08d"
	}
	_, err := db.series.
		Upsert(
			et.Json{
				"tag":      tag,
				"owner_id": ownerId,
				"format":   format,
				"value":    value,
			}).
		Where(Eq("tag", tag)).
		And(Eq("owner_id", ownerId)).
		Exec()
	return err
}

/**
* GetSeries
* @param string tag, ownerId string
* @return (et.Item, error)
**/
func (db *DB) GetSeries(tag, ownerId string) (et.Item, error) {
	result, err := db.series.
		Where(Eq("tag", tag)).
		And(Eq("owner_id", ownerId)).
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
func (db *DB) DeleteSeries(tag, ownerId string) error {
	_, err := db.series.
		Delete().
		Where(Eq("tag", tag)).
		And(Eq("owner_id", ownerId)).
		Exec()
	if err != nil {
		return err
	}
	return nil
}

/**
* NextSeries
* @param string tag, ownerId string
* @return (string, error)
**/
func (db *DB) NextSeries(tag, ownerId string) (string, error) {
	item, err := db.series.
		Update(et.Json{}).
		BeforeUpdate(func(tx *Tx, old, new et.Json) error {
			new["value"] = old["value"].(int) + 1
			return nil
		}).
		Where(Eq("tag", tag)).
		And(Eq("owner_id", ownerId)).
		One()
	if err != nil {
		return "", err
	}
	format := item.String("format")
	value := item.Int("value")
	result := fmt.Sprintf(format, value+1)
	return result, nil
}

/**
* NextValue
* @param string tag, ownerId string
* @return (int, error)
**/
func (db *DB) NextValue(tag, ownerId string) (int, error) {
	item, err := db.series.
		Update(et.Json{}).
		BeforeUpdate(func(tx *Tx, old, new et.Json) error {
			new["value"] = old["value"].(int) + 1
			return nil
		}).
		Where(Eq("tag", tag)).
		And(Eq("owner_id", ownerId)).
		One()
	if err != nil {
		return 0, err
	}
	return item.Int("value"), nil
}
