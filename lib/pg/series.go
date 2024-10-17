package lib

import (
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/linq"
)

/**
* NextSerie return the next value of a serie
* @param tag string
* @return int
* @return error
**/
func (d *Postgres) NextSerie(tag string) (int, error) {
	lock := d.lock(tag)
	db := d.db
	if d.dm != nil {
		db = d.dm
	}

	result, err := nextSerie(db, tag, lock)
	if err != nil {
		return 0, err
	}

	return result, nil
}

/**
* NextCode return the next code of a serie
* @param tag string
* @param format string, example 'A000000-2024' A%05d-2024
* @return string
* @return error
**/
func (d *Postgres) NextCode(tag, format string) (string, error) {
	val, err := d.NextSerie(tag)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf(format, val)

	return result, nil
}

/**
* SetSerie set the value of a serie
* @param tag string
* @param val int
* @return error
**/
func (d *Postgres) SetSerie(tag string, val int) error {
	lock := d.lock(tag)
	db := d.db
	if d.dm != nil {
		db = d.dm
	}

	current, err := currentSerie(db, tag, lock)
	if err != nil {
		return err
	}

	if current == 0 {
		err := insertSerie(db, tag, val, lock)
		if err != nil {
			return err
		}

		return nil
	}

	lock.Lock()
	defer lock.Unlock()

	sql := `
	UPDATE core.SERIES SET
	VALUE = $2
	WHERE SERIE = $1`

	err = db.Exec(sql, tag, val)
	if err != nil {
		return err
	}

	return nil
}

/**
* CurrentSerie return the current value of a serie
* @param tag string
* @return int
**/
func (d *Postgres) CurrentSerie(tag string) (int, error) {
	lock := d.lock(tag)
	db := d.db
	if d.dm != nil {
		db = d.dm
	}

	result, err := currentSerie(db, tag, lock)
	if err != nil {
		return 0, err
	}

	return result, nil
}

/**
* DeleteSerie delete a serie
* @param tag string
* @return int
**/
func (d *Postgres) DeleteSerie(tag string) error {
	lock := d.lock(tag)
	db := d.db
	if d.dm != nil {
		db = d.dm
	}

	err := deleteSerie(db, tag, lock)
	if err != nil {
		return err
	}

	return nil
}

/**
* defineSeries define the series table
* @param db *linq.DB
* @return error
**/
func defineSeries(db *linq.DB) error {
	exists, err := ExistTable(db, "core", "SERIES")
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	sql := `
	CREATE SCHEMA IF NOT EXISTS core;

  CREATE TABLE IF NOT EXISTS core.SERIES(
		SERIE VARCHAR(250) DEFAULT '',
		VALUE BIGINT DEFAULT 0,
		PRIMARY KEY(SERIE)
	);`

	err = db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

/**
* insertSerie insert a serie
* @param db *linq.DB
* @param tag string
* @param val int
* @param lock *sync.RWMutex
* @return error
**/
func insertSerie(db *linq.DB, tag string, val int, lock *sync.RWMutex) error {
	lock.Lock()
	defer lock.Unlock()

	sql := `
	INSERT INTO core.SERIES (SERIE, VALUE)
	VALUES ($1, $2);`

	err := db.Exec(sql, tag, val)
	if err != nil {
		return err
	}

	return nil
}

/**
* currentSerie return the current value of a serie
* @param db *linq.DB
* @param tag string
* @param lock *sync.RWMutex
* @return int
**/
func currentSerie(db *linq.DB, tag string, lock *sync.RWMutex) (int, error) {
	lock.RLock()
	defer lock.RUnlock()

	sql := `
	SELECT VALUE AS result
	FROM core.SERIES
	WHERE SERIE = $1 LIMIT 1;`

	item, err := db.QueryOne(sql, tag)
	if err != nil {
		return 0, err
	}

	if !item.Ok {
		return 0, nil
	}

	result := item.Int("result")

	return result, nil
}

/**
* nextSerie return the next value of a serie
* @param db *linq.DB
* @param tag string
* @param lock *sync.RWMutex
* @return int
**/
func nextSerie(db *linq.DB, tag string, lock *sync.RWMutex) (int, error) {
	current, err := currentSerie(db, tag, lock)
	if err != nil {
		return 0, err
	}

	if current == 0 {
		result := 1
		err := insertSerie(db, tag, result, lock)
		if err != nil {
			return 0, err
		}

		return result, nil
	}

	lock.Lock()
	defer lock.Unlock()

	sql := `
	UPDATE core.SERIES SET	
	VALUE = $2
	WHERE SERIE = $1;`

	result := current + 1
	err = db.Exec(sql, tag, result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

/**
* deleteSerie delete a serie
* @param db *linq.DB
* @param tag string
* @param lock *sync.RWMutex
* @return error
**/
func deleteSerie(db *linq.DB, tag string, lock *sync.RWMutex) error {
	lock.Lock()
	defer lock.Unlock()

	sql := `
	DELETE FROM core.SERIES
	WHERE SERIE = $1;`

	err := db.Exec(sql, tag)
	if err != nil {
		return err
	}

	return nil
}
