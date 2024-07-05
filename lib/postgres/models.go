package lib

import (
	"database/sql"

	"github.com/cgalvisleon/et/et"
)

/**
* Define table models, for the core database
* @param db *sql.DB
* @return string
**/
func defineModels(db *sql.DB) error {
	sql := `
	CREATE SCHEMA IF NOT EXISTS core;

  CREATE TABLE IF NOT EXISTS core.MODELS(
		MAIN VARCHAR(80) DEFAULT '',
    NAME VARCHAR(80) DEFAULT '',
		KIND VARCHAR(80) DEFAULT '',
		VERSION INT DEFAULT 1,
		_DATA JSONB DEFAULT '{}',		
    INDEX SERIAL,
		PRIMARY KEY(MAIN, NAME)
	);  
	CREATE INDEX IF NOT EXISTS MODELS_MAIN_IDX ON core.MODELS(MAIN);
	CREATE INDEX IF NOT EXISTS MODELS_NAME_IDX ON core.MODELS(NAME);
	CREATE INDEX IF NOT EXISTS MODELS_KIND_IDX ON core.MODELS(KIND);
	CREATE INDEX IF NOT EXISTS MODELS_INDEX_IDX ON core.MODELS(INDEX);`

	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

/**
* GetModel get a model from the database
* @param main string
* @param name string
* @return et.Json
**/
func (d *Postgres) GetModel(main, name, kind string) (et.Item, error) {
	sql := `
	SELECT * FROM core.MODELS
	WHERE MAIN = $1 AND NAME = $2 AND KIND = $3;`

	result, err := d.QueryOne(sql, main, name, kind)
	if err != nil {
		return et.Item{}, err
	}

	return result, nil
}

/**
* InsertModel set a model in the database
* @param main string
* @param name string
* @param kind string
* @param version int
* @param data et.Json
* @return error
**/
func (d *Postgres) InsertModel(main, name, kind string, version int, data et.Json) error {
	sql := `
		INSERT INTO core.MODELS (MAIN, NAME, KIND, VERSION, _DATA)
		VALUES ($1, $2, $3, $4, $5);`

	_, err := d.DB.Exec(sql, main, name, kind, version, data)
	if err != nil {
		return err
	}

	return nil
}

/**
* UpdateModel set a model in the database
* @param main string
* @param name string
* @param kind string
* @param version int
* @param data et.Json
* @return error
**/
func (d *Postgres) UpdateModel(main, name, kind string, version int, data et.Json) error {
	sql := `
	UPDATE core.MODELS SET	
	_DATA = $4
	WHERE MAIN = $1 AND NAME = $2 AND KIND = $3;`

	_, err := d.DB.Exec(sql, main, name, kind, data)
	if err != nil {
		return err
	}

	return nil
}

/**
* DeleteModel delete a model from the database
* @param main string
* @param name string
* @param kind string
* @return et.Item
**/
func (d *Postgres) DeleteModel(main, name, kind string) error {
	sql := `
	DELETE FROM core.MODELS
	WHERE MAIN = $1 AND NAME = $2 AND KIND = $3
	RETURNING _DATA INTO result;`

	_, err := d.DB.Query(sql, main, name, kind)
	if err != nil {
		return err
	}

	return nil
}
