package lib

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/linq"
)

/**
* defineSeries define the series table
* @param db *sql.DB
* @return error
**/
func defineSeries(db *sql.DB) error {
	sql := `
	CREATE SCHEMA IF NOT EXISTS core;

  CREATE TABLE IF NOT EXISTS core.SERIES(		
		SERIE VARCHAR(250) DEFAULT '',
		VALUE BIGINT DEFAULT 0,
		PRIMARY KEY(SERIE)
	);`

	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

/**
* insertSerie insert a serie
* @param db *sql.DB
* @param tag string
* @param val int
* @param lock *sync.RWMutex
* @return error
**/
func insertSerie(db *sql.DB, tag string, val int, lock *sync.RWMutex) error {
	lock.Lock()
	defer lock.Unlock()

	sql := `
	INSERT INTO core.SERIES (SERIE, VALUE)
	VALUES ($1, $2);`

	_, err := db.Exec(sql, tag, val)
	if err != nil {
		return err
	}

	return nil
}

/**
* currentSerie return the current value of a serie
* @param db *sql.DB
* @param tag string
* @param lock *sync.RWMutex
* @return int
**/
func currentSerie(db *sql.DB, tag string, lock *sync.RWMutex) (int, error) {
	lock.RLock()
	defer lock.RUnlock()

	sql := `
	SELECT VALUE AS result
	FROM core.SERIES
	WHERE SERIE = $1 LIMIT 1;`

	item, err := linq.QueryOne(db, sql, tag)
	if err != nil {
		return 0, err
	}

	if !item.Ok {
		return 0, nil
	}

	result := item.Int("result")

	return result, nil
}

func nextUUId(db *sql.DB, tag string, lock *sync.RWMutex) (int64, error) {
	now := time.Now()
	result := now.UnixMilli() * 10000
	replica, err := getVarInt(db, "REPLICA", 1)
	if err != nil {
		return 0, err
	}

	if replica < 10 {
		replica = replica * 1000
	} else if replica < 100 {
		replica = replica * 100
	} else {
		replica = replica * 10
	}

	result = result + replica
	key := fmt.Sprintf("%s:%d", tag, result)
	count := cache.Count(key, 1)

	result = result + count
	return result, nil
}

/**
* nextSerie return the next value of a serie
* @param db *sql.DB
* @param tag string
* @param lock *sync.RWMutex
* @return int
**/
func nextSerie(db *sql.DB, tag string, lock *sync.RWMutex) (int, error) {
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
	_, err = db.Exec(sql, tag, result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

/**
* deleteSerie delete a serie
* @param db *sql.DB
* @param tag string
* @param lock *sync.RWMutex
* @return error
**/
func deleteSerie(db *sql.DB, tag string, lock *sync.RWMutex) error {
	lock.Lock()
	defer lock.Unlock()

	sql := `
	DELETE FROM core.SERIES
	WHERE SERIE = $1;`

	_, err := db.Exec(sql, tag)
	if err != nil {
		return err
	}

	return nil
}

/**
* UUIndex return the next value of a serie
* @param tag string
* @return int64
* @return error
**/
func (d *Postgres) UUIndex(tag string) (int64, error) {
	lock := d.Lock(tag)
	result, err := nextUUId(d.DB, tag, lock)
	if err != nil {
		return 0, err
	}

	return result, nil
}

/**
* NextSerie return the next value of a serie
* @param tag string
* @return int
* @return error
**/
func (d *Postgres) NextSerie(tag string) (int, error) {
	lock := d.Lock(tag)
	result, err := nextSerie(d.DB, tag, lock)
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
	lock := d.Lock(tag)
	current, err := currentSerie(d.DB, tag, lock)
	if err != nil {
		return err
	}

	if current == 0 {
		err := insertSerie(d.DB, tag, val, lock)
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

	_, err = d.DB.Exec(sql, tag, val)
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
	lock := d.Lock(tag)
	result, err := currentSerie(d.DB, tag, lock)
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
	lock := d.Lock(tag)
	err := deleteSerie(d.DB, tag, lock)
	if err != nil {
		return err
	}

	return nil
}
