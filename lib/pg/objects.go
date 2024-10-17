package lib

import (
	"github.com/cgalvisleon/et/linq"
)

/**
* defineSync return sql syncs ddl
* @return string
**/
func defineSync(db *linq.DB) error {
	exists, err := ExistTable(db, "core", "OBJECTS")
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	sql := `
  CREATE TABLE IF NOT EXISTS core.OBJECTS(
    DATE_MAKE TIMESTAMP DEFAULT NOW(),
		DATE_UPDATE TIMESTAMP DEFAULT NOW(),
    TABLE_SCHEMA VARCHAR(80) DEFAULT '',
    TABLE_NAME VARCHAR(80) DEFAULT '',
    _IDT VARCHAR(80) DEFAULT '-1',
    INDEX BIGINT DEFAULT 0,
    PRIMARY KEY (TABLE_SCHEMA, TABLE_NAME, _IDT)
  );    
  CREATE INDEX IF NOT EXISTS OBJECTS_TABLE_SCHEMA_IDX ON core.OBJECTS(TABLE_SCHEMA);
  CREATE INDEX IF NOT EXISTS OBJECTS_TABLE_NAME_IDX ON core.OBJECTS(TABLE_NAME);
  CREATE INDEX IF NOT EXISTS OBJECTS__IDT_IDX ON core.OBJECTS(_IDT);  
	CREATE INDEX IF NOT EXISTS OBJECTS_INDEX_IDX ON core.OBJECTS(INDEX);

  CREATE OR REPLACE FUNCTION core.OBJECTS_INSERT()
  RETURNS
    TRIGGER AS $$  
  BEGIN
    IF NEW._IDT = '-1' THEN
      NEW._IDT = uuid_generate_v4();
		END IF;

		INSERT INTO core.OBJECTS(TABLE_SCHEMA, TABLE_NAME, _IDT)
		VALUES (TG_TABLE_SCHEMA, TG_TABLE_NAME, NEW._IDT);

		PERFORM pg_notify(
		'sync',
		json_build_object(
			'schema', TG_TABLE_SCHEMA,
			'table', TG_TABLE_NAME,
			'option', TG_OP,        
			'_idt', NEW._IDT
		)::text
		);
  RETURN NEW;
  END;
  $$ LANGUAGE plpgsql;

	CREATE OR REPLACE FUNCTION core.OBJECTS_UPDATE()
  RETURNS
    TRIGGER AS $$  
  BEGIN
    UPDATE core.OBJECTS SET
		DATE_UPDATE=NOW()
		WHERE _IDT=NEW._IDT;

		PERFORM pg_notify(
		'sync',
		json_build_object(
			'schema', TG_TABLE_SCHEMA,
			'table', TG_TABLE_NAME,
			'option', TG_OP,
			'_idt', NEW._IDT
		)::text
		);
  RETURN NEW;
  END;
  $$ LANGUAGE plpgsql;

  CREATE OR REPLACE FUNCTION core.OBJECTS_DELETE()
  RETURNS
    TRIGGER AS $$  
  BEGIN
    DELETE FROM core.OBJECTS
    WHERE TABLE_SCHEMA = TG_TABLE_SCHEMA
    AND TABLE_NAME = TG_TABLE_NAME
    AND _IDT = OLD._IDT;

		PERFORM pg_notify(
		'sync',
		json_build_object(
			'schema', TG_TABLE_SCHEMA,
			'table', TG_TABLE_NAME,
			'option', TG_OP,
			'_idt', OLD._IDT
		)::text
		);
  RETURN OLD;
  END;
  $$ LANGUAGE plpgsql;`

	err = db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}
