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
func defineModels(db *sql.DB) (string, error) {
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
		return "", err
	}

	return sql, nil
}

/**
* SetModel set a model in the database
* @param main string
* @param name string
* @param kind string
* @param version int
* @param data et.Json
* @return error
**/
func (d *Postgres) UpSertModel(main, name, kind string, version int, data et.Json) (et.Item, error) {
	sql := `
	INSERT INTO core.MODELS AS A (MAIN, NAME, KIND, VERSION, _DATA)
	SELECT $1, $2, $3, $4, $5
	ON CONFLICT (MAIN, NAME) DO UPDATE SET
	_DATA = $5
	RETURNING VERSION INTO result;`

	rows, err := d.DB.Query(sql, main, name, kind, version, data)
	if err != nil {
		return et.Item{}, err
	}

	defer rows.Close()

	return et.Item{
		Ok: true,
		Result: et.Json{
			"message": "Model updated or inserted",
		},
	}, nil
}

/**
* GetModel get a model from the database
* @param main string
* @param name string
* @return et.Json
**/
func (d *Postgres) GetModel(main, name, kind string) (et.Item, error) {
	sql := `
	SELECT _DATA FROM core.MODELS
	WHERE MAIN = $1 AND NAME = $2 AND KIND = $3;`

	rows, err := d.DB.Query(sql, main, name, kind)
	if err != nil {
		return et.Item{}, err
	}

	var result et.Item
	result.OfRows(rows)
	defer rows.Close()

	return result, nil
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
