package lib

import (
	"database/sql"
	"fmt"
)

// ddlSchemes return sql series ddl
func defineSeries(db *sql.DB) (string, error) {
	sql := `
	CREATE SCHEMA IF NOT EXISTS linq;

  CREATE TABLE IF NOT EXISTS core.SERIES(		
		SERIE VARCHAR(250) DEFAULT '',
		VALUE BIGINT DEFAULT 0,
		PRIMARY KEY(SERIE)
	);`

	_, err := db.Exec(sql)
	if err != nil {
		return "", err
	}

	return sql, nil
}

/**
* NextSerie return the next value of a serie
* @param tag string
* @return string
**/
func (d *Postgres) NextSerie(tag string) int64 {
	sql := `
	INSERT INTO core.SERIES AS A (SERIE, VALUE)
	SELECT $1, 1
	ON CONFLICT (SERIE) DO UPDATE SET
	VALUE = A.VALUE + 1
	RETURNING VALUE INTO result;`

	rows, err := d.DB.Query(sql, tag)
	if err != nil {
		return -1
	}

	var result int64
	for rows.Next() {
		rows.Scan(&result)
	}
	defer rows.Close()

	return result
}

/**
* NextCode return the next code of a serie
* @param tag string
* @param format string, example 'A000000-2024' A%05d-2024
* @return string
**/
func (d *Postgres) NextCode(tag, format string) string {
	val := d.NextSerie(tag)
	result := fmt.Sprintf(format, val)

	return result
}

/**
* SetSerie set the value of a serie
* @param tag string
* @param val int
* @return int
**/
func (d *Postgres) SetSerie(tag string, val int64) int64 {
	sql := `
	INSERT INTO core.SERIES AS A (SERIE, VALUE)
	SELECT $1, $2
	ON CONFLICT (SERIE) DO UPDATE SET
	VALUE = $2
	RETURNING VALUE INTO result;`

	rows, err := d.DB.Query(sql, tag, val)
	if err != nil {
		return 0
	}

	var result int64
	for rows.Next() {
		rows.Scan(&result)
	}
	defer rows.Close()

	return result
}

/**
* CurrentSerie return the current value of a serie
* @param tag string
* @return int
**/
func (d *Postgres) CurrentSerie(tag string) int64 {
	sql := `
	SELECT VALUE INTO result
	FROM core.SERIES
	WHERE SERIE = $1 LIMIT 1;`

	rows, err := d.DB.Query(sql, tag)
	if err != nil {
		return 0
	}

	var result int64
	for rows.Next() {
		rows.Scan(&result)
	}
	defer rows.Close()

	return result
}

/**
* DeleteSerie delete a serie
* @param tag string
* @return int
**/
func (d *Postgres) DeleteSerie(tag string) int64 {
	sql := `
	DELETE FROM core.SERIES
	WHERE SERIE = $1
	RETURNING VALUE INTO result;`

	rows, err := d.DB.Query(sql, tag)
	if err != nil {
		return 0
	}

	var result int64
	for rows.Next() {
		rows.Scan(&result)
	}
	defer rows.Close()

	return result
}
