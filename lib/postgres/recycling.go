package lib

import "database/sql"

/**
* defineRecicle create the recicle schema
* @param db *sql.DB
* @return string
* @return error
**/
func defineRecycling(db *sql.DB) error {
	sql := `
	CREATE SCHEMA IF NOT EXISTS core;

  CREATE TABLE IF NOT EXISTS core.RECYCLING(
    DATE_MAKE TIMESTAMP DEFAULT NOW(),
    TABLE_SCHEMA VARCHAR(80) DEFAULT '',
    TABLE_NAME VARCHAR(80) DEFAULT '',
    _IDT VARCHAR(80) DEFAULT '-1',
    INDEX SERIAL,
    PRIMARY KEY (TABLE_SCHEMA, TABLE_NAME, _IDT)
  );    
  CREATE INDEX IF NOT EXISTS RECYCLING_TABLE_SCHEMA_IDX ON core.RECYCLING(TABLE_SCHEMA);
  CREATE INDEX IF NOT EXISTS RECYCLING_TABLE_NAME_IDX ON core.RECYCLING(TABLE_NAME);
  CREATE INDEX IF NOT EXISTS RECYCLING__IDT_IDX ON core.RECYCLING(_IDT);
	CREATE INDEX IF NOT EXISTS RECYCLING_INDEX_IDX ON core.RECYCLING(INDEX);  

  CREATE OR REPLACE FUNCTION core.RECYCLING_UPDATE()
  RETURNS
    TRIGGER AS $$
  DECLARE
    CHANNEL VARCHAR(250);
  BEGIN
    IF NEW._STATE != OLD._STATE && NEW._STATE == '-2' THEN      
      INSERT INTO core.RECYCLING(TABLE_SCHEMA, TABLE_NAME, _IDT)
      VALUES (TG_TABLE_SCHEMA, TG_TABLE_NAME, NEW._IDT);

      PERFORM pg_notify(
      'recycling',
      json_build_object(
        '_idt', NEW._IDT
      )::text
      );
		ELSEIF NEW._STATE != OLD._STATE THEN
      DELETE FROM core.RECYCLING
      WHERE TABLE_SCHEMA = TG_TABLE_SCHEMA
      AND TABLE_NAME = TG_TABLE_NAME
      AND _IDT = NEW._IDT;
    END IF;

  RETURN NEW;
  END;
  $$ LANGUAGE plpgsql;`

	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}
