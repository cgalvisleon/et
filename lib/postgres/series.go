package lib

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/linq"
)

// ddlSchemes return sql series ddl
func defineSeries(db *sql.DB) error {
	sql := `
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
func insertSerie(db *sql.DB, tag string, val int, lock *sync.RWMutex) (int, error) {
	lock.Lock()

	sql := `
	INSERT INTO core.SERIES (SERIE, VALUE)
	VALUES ($1, $2);`

	_, err := db.Exec(sql, tag, val)
	if err != nil {
		return 0, err
	}

	lock.Unlock()

	return val, nil
}

/**
* currentSerie return the current value of a serie
* @param db *sql.DB
* @param tag string
* @param lock *sync.RWMutex
* @return int
**/
func currentSerie(db *sql.DB, tag string, lock *sync.RWMutex) (int, error) {
	defer lock.RUnlock()
	lock.RLock()

	sql := `
	SELECT VALUE INTO result
	FROM core.SERIES
	WHERE SERIE = $1 LIMIT 1;`

	rows, err := db.Query(sql, tag)
	if err != nil {
		return 0, err
	}

	defer rows.Close()

	item := linq.RowsItem(rows)
	if !item.Ok {
		result, err := insertSerie(db, tag, 1, lock)
		if err != nil {
			return 0, err
		}

		return result, nil
	}

	result := item.Int("result")

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

	lock.Lock()
	sql := `
	UPDATE core.SERIES SET	
	VALUE = $2
	WHERE SERIE = $1
	RETERNING VALUE INTO result;`

	result := current + 1
	_, err = db.Exec(sql, tag, result)
	if err != nil {
		return 0, err
	}

	lock.Unlock()

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

	sql := `
	DELETE FROM core.SERIES
	WHERE SERIE = $1
	RETURNING VALUE INTO result;`

	_, err := db.Exec(sql, tag)
	if err != nil {
		return err
	}

	lock.Unlock()

	return nil
}

/**
* NextSerie return the next value of a serie
* @param tag string
* @return int
* @return error
**/
func (d *Postgres) NextSerie(tag string) (int, error) {
	lock := d.lock(tag)
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
	lock := d.lock(tag)
	_, err := currentSerie(d.DB, tag, lock)
	if err != nil {
		return err
	}

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
	lock := d.lock(tag)
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
	lock := d.lock(tag)
	err := deleteSerie(d.DB, tag, lock)
	if err != nil {
		return err
	}

	return nil
}
