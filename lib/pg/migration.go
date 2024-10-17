package lib

import (
	"github.com/cgalvisleon/et/linq"
)

/**
* UpSertMigrateId upsert a migrate id
* @param old_id string
* @param _id string
* @param tag string
* @return error
**/
func (d *Postgres) UpSertMigrateId(old_id, _id, tag string) error {
	return upSertMigrateId(d.db, old_id, _id, tag)
}

/**
* GetMigrateId get a migrate id
* @param old_id string
* @param tag string
* @return string
* @return error
**/
func (d *Postgres) GetMigrateId(old_id, tag string) (string, error) {
	return getMigrateId(d.db, old_id, tag)
}

func defineMigrateId(db *linq.DB) error {
	exists, err := ExistTable(db, "core", "MIGRATES")
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	sql := `
	CREATE TABLE IF NOT EXISTS core.MIGRATES(    
    OLD_ID VARCHAR(80) DEFAULT '-1',
		_ID VARCHAR(80) DEFAULT '-1',
		TAG VARCHAR(80) DEFAULT '-1',
    INDEX SERIAL,
    PRIMARY KEY (OLD_ID, TAG)
  );    
  CREATE INDEX IF NOT EXISTS MIGRATE_ID_OLD_ID_IDX ON core.MIGRATES(OLD_ID);
	CREATE INDEX IF NOT EXISTS MIGRATE_ID__ID_IDX ON core.MIGRATES(OLD_ID);
	CREATE INDEX IF NOT EXISTS MIGRATE_ID_TAG_IDX ON core.MIGRATES(OLD_ID);
	CREATE INDEX IF NOT EXISTS MIGRATE_ID_INDEX_IDX ON core.MIGRATES(OLD_ID);`

	err = db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

func insertMigrateId(db *linq.DB, old_id, id, tag string) error {
	sql := `
	INSERT INTO core.MIGRATES(OLD_ID, _ID, TAG)
	VALUES($1, $2, $3);`

	err := db.Exec(sql, old_id, id, tag)
	if err != nil {
		return err
	}

	return nil
}

func upSertMigrateId(db *linq.DB, old_id, id, tag string) error {
	sql := `
	SELECT
	_ID
	FROM core.MIGRATES
	WHERE OLD_ID = $1
	AND TAG = $2`

	items, err := db.Query(sql, old_id, tag)
	if err != nil {
		return err
	}

	if items.Count == 0 {
		return insertMigrateId(db, old_id, id, tag)
	}

	sql = `
	UPDATE core.MIGRATES
	SET _ID = $1
	WHERE OLD_ID = $2
	AND TAG = $3;`

	err = db.Exec(sql, id, old_id, tag)
	if err != nil {
		return err
	}

	return nil
}

func getMigrateId(db *linq.DB, old_id, tag string) (string, error) {
	sql := `
	SELECT
	_ID
	FROM core.MIGRATES
	WHERE OLD_ID = $1
	AND TAG = $2`

	items, err := db.QueryOne(sql, old_id, tag)
	if err != nil {
		return "", err
	}

	if !items.Ok {
		return old_id, nil
	}

	result := items.Key("_id")

	return result, nil
}
