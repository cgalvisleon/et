package lib

import "database/sql"

/**
* defineSync return sql syncs ddl
* @return string
**/
func defineSync(db *sql.DB) error {
	sql := `
  CREATE TABLE IF NOT EXISTS core.SYNCS(
    DATE_MAKE TIMESTAMP DEFAULT NOW(),
    DATE_UPDATE TIMESTAMP DEFAULT NOW(),
    TABLE_SCHEMA VARCHAR(80) DEFAULT '',
    TABLE_NAME VARCHAR(80) DEFAULT '',
    _IDT VARCHAR(80) DEFAULT '-1',
    ACTION VARCHAR(80) DEFAULT '',
    _SYNC BOOLEAN DEFAULT FALSE,    
    INDEX SERIAL,
    PRIMARY KEY (TABLE_SCHEMA, TABLE_NAME, _IDT)
  );    
  CREATE INDEX IF NOT EXISTS SYNCS_TABLE_SCHEMA_IDX ON core.SYNCS(TABLE_SCHEMA);
  CREATE INDEX IF NOT EXISTS SYNCS_TABLE_NAME_IDX ON core.SYNCS(TABLE_NAME);
  CREATE INDEX IF NOT EXISTS SYNCS__IDT_IDX ON core.SYNCS(_IDT);
  CREATE INDEX IF NOT EXISTS SYNCS_ACTION_IDX ON core.SYNCS(ACTION);
  CREATE INDEX IF NOT EXISTS SYNCS__SYNC_IDX ON core.SYNCS(_SYNC);
	CREATE INDEX IF NOT EXISTS SYNCS_INDEX_IDX ON core.SYNCS(INDEX);

  CREATE OR REPLACE FUNCTION core.SYNC_INSERT()
  RETURNS
    TRIGGER AS $$
  DECLARE
   CHANNEL VARCHAR(250);
  BEGIN
    IF NEW._IDT = '-1' THEN
      NEW._IDT = uuid_generate_v4();

      INSERT INTO core.SYNCS(TABLE_SCHEMA, TABLE_NAME, _IDT, ACTION)
      VALUES (TG_TABLE_SCHEMA, TG_TABLE_NAME, NEW._IDT, TG_OP);

      PERFORM pg_notify(
      'sync',
      json_build_object(
        'option', TG_OP,        
        '_idt', NEW._IDT
      )::text
      );
    END IF;

  RETURN NEW;
  END;
  $$ LANGUAGE plpgsql;

  CREATE OR REPLACE FUNCTION core.SYNC_UPDATE()
  RETURNS
    TRIGGER AS $$
  DECLARE
    CHANNEL VARCHAR(250);
  BEGIN
    IF NEW._IDT = '-1' AND OLD._IDT != '-1' THEN
      NEW._IDT = OLD._IDT;
    ELSE
     IF NEW._IDT = '-1' THEN
       NEW._IDT = uuid_generate_v4();
     END IF;
     INSERT INTO core.SYNCS(TABLE_SCHEMA, TABLE_NAME, _IDT, ACTION)
     VALUES (TG_TABLE_SCHEMA, TG_TABLE_NAME, NEW._IDT, TG_OP)
		 ON CONFLICT(TABLE_SCHEMA, TABLE_NAME, _IDT) DO UPDATE SET
     DATE_UPDATE = NOW(),
     ACTION = TG_OP,
     _SYNC = FALSE;

     PERFORM pg_notify(
     'sync',
     json_build_object(
       'option', TG_OP,
       '_idt', NEW._IDT
     )::text
     );     
    END IF; 

  RETURN NEW;
  END;
  $$ LANGUAGE plpgsql;

  CREATE OR REPLACE FUNCTION core.SYNC_DELETE()
  RETURNS
    TRIGGER AS $$
  DECLARE
    VINDEX INTEGER;
    CHANNEL VARCHAR(250);
  BEGIN
    SELECT INDEX INTO VINDEX
    FROM core.SYNCS
    WHERE TABLE_SCHEMA = TG_TABLE_SCHEMA
    AND TABLE_NAME = TG_TABLE_NAME
    AND _IDT = OLD._IDT
    LIMIT 1;
    IF FOUND THEN
      UPDATE core.SYNCS SET
      DATE_UPDATE = NOW(),
      ACTION = TG_OP,
      _SYNC = FALSE
      WHERE INDEX = VINDEX;
      
      PERFORM pg_notify(
      'sync',
      json_build_object(
        'option', TG_OP,
        '_idt', OLD._IDT
      )::text
      );      
    END IF;

  RETURN OLD;
  END;
  $$ LANGUAGE plpgsql;`

	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}
