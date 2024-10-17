package lib

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
)

/**
* defineCore create the core schema
* @param db *linq.DB
* @return error
**/
func defineCore(db *linq.DB) error {
	sql := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	CREATE EXTENSION IF NOT EXISTS pgcrypto;
	CREATE SCHEMA IF NOT EXISTS core;

  CREATE TABLE IF NOT EXISTS core.COMMAND(		
		_ID VARCHAR(80) DEFAULT '-1',
		SQL BYTEA,
		MUTEX INT DEFAULT 0,
		INDEX BIGINT DEFAULT 0,
		PRIMARY KEY(_ID)
	);
	CREATE INDEX IF NOT EXISTS COMMAND_INDEX_IDX ON core.COMMAND(INDEX);

	CREATE OR REPLACE FUNCTION core.COMMAND_INSERT()
  RETURNS
    TRIGGER AS $$  
  BEGIN
	 	IF NEW.MUTEX = 0 THEN
			PERFORM pg_notify(
			'command',
			json_build_object(        
				'_id', NEW._ID
			)::text
			);
		END IF;
  RETURN NEW;
  END;
  $$ LANGUAGE plpgsql;

	DROP TRIGGER IF EXISTS COMMAND_INSERT ON core.COMMAND CASCADE;
	CREATE TRIGGER COMMAND_INSERT
	BEFORE INSERT ON core.COMMAND
	FOR EACH ROW
	EXECUTE PROCEDURE core.COMMAND_INSERT();`

	_, err := db.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

/**
* SetCommand
* @params query string
**/
func (d *Postgres) SetCommand(query string) error {
	sql := `
	INSERT INTO core.COMMAND (_ID, SQL, MUTEX, INDEX)
	VALUES ($1, $2, $3, $4);`

	id := utility.UUID()
	index := utility.UUIndex("commnad")
	_, err := d.db.DB.Exec(sql, id, []byte(query), 0, index)
	if err != nil {
		logs.Debug(et.Json{
			"_id":   id,
			"sql":   query,
			"index": index,
		}.ToString())
		return err
	}

	if d.lastcomand < index {
		d.lastcomand = index
	}

	return nil
}

/**
* SetMutex
* @params id string
* @params query string
* @params index int64
* @return error
**/
func (d *Postgres) SetMutex(id, query string, index int64) error {
	sql := `
	SELECT INDEX
	FROM core.COMMAND
	WHERE _ID = $1;`

	item, err := d.db.QueryOne(sql, id)
	if err != nil {
		return err
	}

	if item.Ok {
		return nil
	}

	sql = `
	INSERT INTO core.COMMAND (_ID, SQL, MUTEX, INDEX)
	VALUES ($1, $2, $3, $4);`

	_, err = d.db.DB.Exec(sql, id, []byte(query), 1, index)
	if err != nil {
		return err
	}

	if d.lastcomand < index {
		d.lastcomand = index
	}

	return nil
}

/**
* GetCommand
* @params id string
* @return et.Item
* @return error
**/
func (d *Postgres) GetCommand(id string) (et.Item, error) {
	var result et.Item = et.Item{}

	query := `
	SELECT _ID, SQL, INDEX
	FROM core.COMMAND
	WHERE _ID = $1 LIMIT 1;`

	rows, err := d.db.DB.Query(query, id)
	if err != nil {
		return result, err
	}

	var _id string
	var sql []byte
	var index int64
	for rows.Next() {
		rows.Scan(&_id, &sql, &index)
		result = et.Item{
			Ok: true,
			Result: et.Json{
				"_id":   _id,
				"sql":   string(sql),
				"index": index,
			},
		}
	}

	return result, nil
}

/**
* getLastCommand
* @return int64
* @return error
**/
func (d *Postgres) getLastCommand() (int64, error) {
	var result int64 = 0

	sql := `
	SELECT MAX(INDEX) AS result
	FROM core.COMMAND;`

	rows, err := d.db.DB.Query(sql)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		rows.Scan(&result)
	}

	return result, nil
}

/**
* SyncCommand
* @return error
**/
func (d *Postgres) SyncCommand() error {
	if d.dm == nil {
		return logs.Alertm(ERR_MASTER_DB_NOT_FOUND)
	}

	var ok bool = true
	var page int = 1
	var rows int = 1000
	var total int = 0
	lastIndex, err := d.getLastCommand()
	if err != nil {
		return err
	}
	logs.Info(`Sync commands`)

	for ok {
		ok = false

		logs.Debug("Sync commands page:", page)

		offset := (page - 1) * rows
		sql := `
		SELECT A._ID, A.SQL, A.INDEX
		FROM core.COMMAND A
		WHERE A.INDEX>=$3
		ORDER BY A.index
		LIMIT $1 OFFSET $2;`

		rows, err := d.dm.DB.Query(sql, rows, offset, lastIndex)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item et.Item
			item.ScanRows(rows)
			id := item.Str("_id")
			sql := item.Str("sql")
			lastIndex = item.Int64("index")

			err = d.SetMutex(id, sql, lastIndex)
			if err != nil {
				return err
			} else {
				total++
			}

			ok = true
		}

		page++
	}

	logs.Infof(`Sync commands total: %d`, total)

	return nil
}
