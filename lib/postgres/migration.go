package lib

import (
	"database/sql"

	"github.com/cgalvisleon/et/linq"
)

func defineMigrateId(db *sql.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS core.MIGRATE_ID(    
    OLD_ID VARCHAR(80) DEFAULT '-1',
		_ID VARCHAR(80) DEFAULT '-1',
		TAG VARCHAR(80) DEFAULT '-1',
    INDEX SERIAL,
    PRIMARY KEY (OLD_ID, TAG)
  );    
  CREATE INDEX IF NOT EXISTS MIGRATE_ID_OLD_ID_IDX ON core.MIGRATE_ID(OLD_ID);
	CREATE INDEX IF NOT EXISTS MIGRATE_ID__ID_IDX ON core.MIGRATE_ID(OLD_ID);
	CREATE INDEX IF NOT EXISTS MIGRATE_ID_TAG_IDX ON core.MIGRATE_ID(OLD_ID);
	CREATE INDEX IF NOT EXISTS MIGRATE_ID_INDEX_IDX ON core.MIGRATE_ID(OLD_ID);`

	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

func insertMigrateId(db *sql.DB, old_id, tag string) error {
	sql := `
	INSERT INTO core.MIGRATE_ID(OLD_ID, _ID, TAG)
	VALUES($1, $2, $3);`

	_, err := db.Exec(sql, old_id, old_id, tag)
	if err != nil {
		return err
	}

	return nil
}

func upSertMigrateId(db *sql.DB, old_id, _id, tag string) error {
	sql := `
	SELECT
	_ID
	FROM core.MIGRATE_ID
	WHERE OLD_ID = $1
	AND TAG = $2`

	rows, err := db.Query(sql, old_id, tag)
	if err != nil {
		return err
	}
	defer rows.Close()

	items := linq.RowsItems(rows)
	if items.Count == 0 {
		return insertMigrateId(db, old_id, tag)
	}

	sql = `
	UPDATE core.MIGRATE_ID
	SET _ID = $1
	WHERE OLD_ID = $2
	AND TAG = $3;`

	_, err = db.Exec(sql, _id, old_id, tag)
	if err != nil {
		return err
	}

	return nil
}

func getMigrateId(db *sql.DB, old_id, tag string) (string, error) {
	sql := `
	SELECT
	_ID
	FROM core.MIGRATE_ID
	WHERE OLD_ID = $1
	AND TAG = $2`

	rows, err := db.Query(sql, old_id, tag)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	items := linq.RowsItems(rows)
	if items.Count == 0 {
		err = insertMigrateId(db, old_id, tag)
		if err != nil {
			return "", err
		}

		return old_id, nil
	}

	item := items.Result[0]
	return item.Key("_id"), nil
}
