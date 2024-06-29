package lib

import (
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/strs"
)

// DDL Data Definition Language
// This package contains the functions to definition data elements in the database

// ddlSchemes return sql series ddl
func ddlSeries() string {
	return `
	-- DROP TABLE IF EXISTS linq.SERIES CASCADE;
	CREATE SCHEMA IF NOT EXISTS linq;

  CREATE TABLE IF NOT EXISTS linq.SERIES(		
		SERIE VARCHAR(250) DEFAULT '',
		VALUE BIGINT DEFAULT 0,
		PRIMARY KEY(SERIE)
	);
	
	CREATE OR REPLACE FUNCTION linq.nextserie(tag VARCHAR(250))
	RETURNS BIGINT AS $$
	DECLARE
	 result BIGINT;
	BEGIN
	 INSERT INTO linq.SERIES AS A (SERIE, VALUE)
	 SELECT tag, 1
	 ON CONFLICT (SERIE) DO UPDATE SET
	 VALUE = A.VALUE + 1
	 RETURNING VALUE INTO result;

	 RETURN COALESCE(result, 0);
	END;
	$$ LANGUAGE plpgsql;
	
	CREATE OR REPLACE FUNCTION linq.setserie(tag VARCHAR(250), val BIGINT)
	RETURNS BIGINT AS $$
	DECLARE
	 result BIGINT;
	BEGIN
	 INSERT INTO linq.SERIES AS A (SERIE, VALUE)
	 SELECT tag, val
	 ON CONFLICT (SERIE) DO UPDATE SET
	 VALUE = val
	 WHERE A.VALUE < val
	 RETURNING VALUE INTO result;

	 RETURN COALESCE(result, 0);
	END;
	$$ LANGUAGE plpgsql;
	
	CREATE OR REPLACE FUNCTION linq.currserie(tag VARCHAR(250))
	RETURNS BIGINT AS $$
	DECLARE
	 result BIGINT;
	BEGIN
	 SELECT VALUE INTO result
	 FROM linq.SERIES
	 WHERE SERIE = tag LIMIT 1;

	 RETURN COALESCE(result, 0);
	END;
	$$ LANGUAGE plpgsql;

	CREATE OR REPLACE FUNCTION linq.delserie(tag VARCHAR(250))
	RETURNS BIGINT AS $$
	DECLARE
	 result BIGINT;
	BEGIN
	 DELETE FROM linq.SERIES
	 WHERE SERIE = tag
	 RETURNING VALUE INTO result;

	 RETURN COALESCE(result, 0);
	END;
	$$ LANGUAGE plpgsql;

	/* TRIGGER FUNTION */
	CREATE OR REPLACE FUNCTION linq.SERIES_INSERT()
	RETURNS
	TRIGGER AS $$
	DECLARE
	VINDEX BIGINT;
	VTAG VARCHAR(250);
	BEGIN
	IF NEW._INDEX = 0 THEN
		VTAG = TG_TABLE_SCHEMA || '.' || TG_TABLE_NAME;
		SELECT linq.nextserie(VTAG) INTO VINDEX;
		NEW._INDEX = VINDEX;
	END IF;

	RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;

	CREATE OR REPLACE FUNCTION linq.SERIES_UPDATE()
	RETURNS
	TRIGGER AS $$
	DECLARE
	VINDEX BIGINT;
	VTAG VARCHAR(250);
	BEGIN
	IF NEW._INDEX > OLD._INDEX THEN
		VTAG = TG_TABLE_SCHEMA || '.' || TG_TABLE_NAME;
		PERFORM linq.setserie(VTAG, NEW._INDEX);
	END IF;

	RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;
	`
}

// ddlListen return sql used listen ddl
func ddlListen() string {
	return `
	CREATE SCHEMA IF NOT EXISTS linq;

	CREATE OR REPLACE FUNCTION linq.SYNC_INSERT()
  RETURNS
    TRIGGER AS $$
  DECLARE
   CHANNEL VARCHAR(250);
  BEGIN
    IF NEW._IDT = '-1' THEN
      NEW._IDT = uuid_generate_v4();
		END IF;

		CHANNEL = TG_TABLE_SCHEMA || '.' || TG_TABLE_NAME;
		PERFORM pg_notify(
		CHANNEL,
		json_build_object(
			'option', TG_OP,
			'_idt', NEW._IDT
		)::text
		);
    
  RETURN NEW;
  END;
  $$ LANGUAGE plpgsql;

  CREATE OR REPLACE FUNCTION linq.SYNC_UPDATE()
  RETURNS
    TRIGGER AS $$
  DECLARE
    CHANNEL VARCHAR(250);
  BEGIN
    IF NEW._IDT = '-1' THEN
			NEW._IDT = uuid_generate_v4();    
    END IF;
    
		CHANNEL = TG_TABLE_SCHEMA || '.' || TG_TABLE_NAME;
		PERFORM pg_notify(
		CHANNEL,
		json_build_object(
			'option', TG_OP,
			'_idt', NEW._IDT  
		)::text
		);

  RETURN NEW;
  END;
  $$ LANGUAGE plpgsql;

  CREATE OR REPLACE linq.FUNCTION SYNC_DELETE()
  RETURNS
    TRIGGER AS $$
  DECLARE
    CHANNEL VARCHAR(250);
  BEGIN
		CHANNEL = TG_TABLE_SCHEMA || '.' || TG_TABLE_NAME;
		PERFORM pg_notify(
		CHANNEL,
		json_build_object(
			'option', TG_OP,
			'_idt', OLD._IDT
		)::text
		);

  RETURN OLD;
  END;
  $$ LANGUAGE plpgsql;
	`
}

// ddlSync return sql sync ddl
func ddlSync() string {
	return `
  -- DROP TABLE IF EXISTS SYNCS CASCADE;
	CREATE SCHEMA IF NOT EXISTS linq;

  CREATE TABLE IF NOT EXISTS linq.SYNCS(
    DATE_MAKE TIMESTAMP DEFAULT NOW(),
    DATE_UPDATE TIMESTAMP DEFAULT NOW(),
    TABLE_SCHEMA VARCHAR(80) DEFAULT '',
    TABLE_NAME VARCHAR(80) DEFAULT '',
    _IDT VARCHAR(80) DEFAULT '-1',
    ACTION VARCHAR(80) DEFAULT '',
    _SYNC BOOLEAN DEFAULT FALSE,    
    _INDEX BIGINT DEFAULT 0,
    PRIMARY KEY (TABLE_SCHEMA, TABLE_NAME, _IDT)
  );  
  CREATE INDEX IF NOT EXISTS SYNCS__INDEX_IDX ON linq.SYNCS(_INDEX);
  CREATE INDEX IF NOT EXISTS SYNCS_TABLE_SCHEMA_IDX ON linq.SYNCS(TABLE_SCHEMA);
  CREATE INDEX IF NOT EXISTS SYNCS_TABLE_NAME_IDX ON linq.SYNCS(TABLE_NAME);
  CREATE INDEX IF NOT EXISTS SYNCS__IDT_IDX ON linq.SYNCS(_IDT);
  CREATE INDEX IF NOT EXISTS SYNCS_ACTION_IDX ON linq.SYNCS(ACTION);
  CREATE INDEX IF NOT EXISTS SYNCS__SYNC_IDX ON linq.SYNCS(_SYNC);

  CREATE OR REPLACE FUNCTION linq.SYNC_INSERT()
  RETURNS
    TRIGGER AS $$
  DECLARE
   CHANNEL VARCHAR(250);
  BEGIN
    IF NEW._IDT = '-1' THEN
      NEW._IDT = uuid_generate_v4();

      INSERT INTO linq.SYNCS(TABLE_SCHEMA, TABLE_NAME, _IDT, ACTION)
      VALUES (TG_TABLE_SCHEMA, TG_TABLE_NAME, NEW._IDT, TG_OP);

      PERFORM pg_notify(
      'sync',
      json_build_object(
        'option', TG_OP,        
        '_idt', NEW._IDT
      )::text
      );

      CHANNEL = TG_TABLE_SCHEMA || '.' || TG_TABLE_NAME;
      PERFORM pg_notify(
      CHANNEL,
      json_build_object(
        'option', TG_OP,
        '_idt', NEW._IDT
      )::text
      );
    END IF;

  RETURN NEW;
  END;
  $$ LANGUAGE plpgsql;

  CREATE OR REPLACE FUNCTION linq.SYNC_UPDATE()
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
     INSERT INTO linq.SYNCS(TABLE_SCHEMA, TABLE_NAME, _IDT, ACTION)
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

     CHANNEL = TG_TABLE_SCHEMA || '.' || TG_TABLE_NAME;
     PERFORM pg_notify(
     CHANNEL,
     json_build_object(
       'option', TG_OP,
       '_idt', NEW._IDT  
     )::text
     );
    END IF; 

  RETURN NEW;
  END;
  $$ LANGUAGE plpgsql;

  CREATE OR REPLACE FUNCTION linq.SYNC_DELETE()
  RETURNS
    TRIGGER AS $$
  DECLARE
    VINDEX INTEGER;
    CHANNEL VARCHAR(250);
  BEGIN
    SELECT _INDEX INTO VINDEX
    FROM linq.SYNCS
    WHERE TABLE_SCHEMA = TG_TABLE_SCHEMA
    AND TABLE_NAME = TG_TABLE_NAME
    AND _IDT = OLD._IDT
    LIMIT 1;
    IF FOUND THEN
      UPDATE linq.SYNCS SET
      DATE_UPDATE = NOW(),
      ACTION = TG_OP,
      _SYNC = FALSE
      WHERE _INDEX = VINDEX;
      
      PERFORM pg_notify(
      'sync',
      json_build_object(
        'option', TG_OP,
        '_idt', OLD._IDT
      )::text
      );

      CHANNEL = TG_TABLE_SCHEMA || '.' || TG_TABLE_NAME;
      PERFORM pg_notify(
      CHANNEL,
      json_build_object(
        'option', TG_OP,
        '_idt', OLD._IDT
      )::text
      );      
    END IF;

  RETURN OLD;
  END;
  $$ LANGUAGE plpgsql;

	DROP TRIGGER IF EXISTS SERIES_INSERT ON linq.SYNCS CASCADE;
	CREATE TRIGGER SERIES_INSERT
	BEFORE INSERT ON linq.SYNCS
	FOR EACH ROW
	EXECUTE PROCEDURE linq.SERIES_INSERT();

	DROP TRIGGER IF EXISTS SERIES_UPDATE ON linq.SYNCS CASCADE;
	CREATE TRIGGER SERIES_UPDATE
	AFTER UPDATE ON linq.SYNCS
	FOR EACH ROW WHEN (NEW!=OLD)
	EXECUTE PROCEDURE linq.SERIES_UPDATE();
	`
}

// ddlRecicling return sql recicling ddl
func ddlRecycling() string {
	return `
	-- DROP TABLE IF EXISTS linq.RECYCLING CASCADE;
	CREATE SCHEMA IF NOT EXISTS linq;

  CREATE TABLE IF NOT EXISTS linq.RECYCLING(
		TABLE_SCHEMA VARCHAR(80) DEFAULT '',
    TABLE_NAME VARCHAR(80) DEFAULT '',
    _IDT VARCHAR(80) DEFAULT '-1',
    _INDEX BIGINT DEFAULT 0,
		PRIMARY KEY(TABLE_SCHEMA, TABLE_NAME, _IDT)
	);
  CREATE INDEX IF NOT EXISTS RECYCLING__INDEX_IDX ON linq.RECYCLING(_INDEX);
	CREATE INDEX IF NOT EXISTS RECYCLING_TABLE_SCHEMA_IDX ON linq.RECYCLING(TABLE_SCHEMA);
	CREATE INDEX IF NOT EXISTS RECYCLING_TABLE_NAME_IDX ON linq.RECYCLING(TABLE_NAME);
	CREATE INDEX IF NOT EXISTS RECYCLING__IDT_IDX ON linq.RECYCLING(_IDT);

	CREATE OR REPLACE FUNCTION linq.RECYCLING()
  RETURNS
    TRIGGER AS $$
  BEGIN
		IF NEW._STATE != OLD._STATE AND NEW._STATE = '-2' THEN
    	INSERT INTO linq.RECYCLING(TABLE_SCHEMA, TABLE_NAME, _IDT)
    	VALUES (TG_TABLE_SCHEMA, TG_TABLE_NAME, NEW._IDT);

      PERFORM pg_notify(
      'recycling',
      json_build_object(
        '_idt', NEW._IDT
      )::text
      );
		ELSEIF NEW._STATE != OLD._STATE AND OLD._STATE = '-2' THEN
			DELETE FROM linq.RECYCLING WHERE _IDT=NEW._IDT;
    END IF;

  RETURN NEW;
  END;
  $$ LANGUAGE plpgsql;

	CREATE OR REPLACE FUNCTION linq.ERASE()
  RETURNS
    TRIGGER AS $$
  BEGIN
		DELETE FROM linq.RECYCLING WHERE _IDT=OLD._IDT;
  	RETURN OLD;
  END;
  $$ LANGUAGE plpgsql;

	DROP TRIGGER IF EXISTS SERIES_INSERT ON linq.RECYCLING CASCADE;
	CREATE TRIGGER SERIES_INSERT
	BEFORE INSERT ON linq.RECYCLING
	FOR EACH ROW
	EXECUTE PROCEDURE linq.SERIES_INSERT();

	DROP TRIGGER IF EXISTS SERIES_UPDATE ON linq.RECYCLING CASCADE;
	CREATE TRIGGER SERIES_UPDATE
	AFTER UPDATE ON linq.RECYCLING
	FOR EACH ROW WHEN (NEW!=OLD)
	EXECUTE PROCEDURE linq.SERIES_UPDATE();
	`
}

func ddlModels() string {
	return `
	-- DROP TABLE IF EXISTS linq.MODELS CASCADE;
	CREATE SCHEMA IF NOT EXISTS linq;

  CREATE TABLE IF NOT EXISTS linq.MODELS(
		TABLE_SCHEMA VARCHAR(80) DEFAULT '',
    TABLE_NAME VARCHAR(80) DEFAULT '',
		DEFINTION JSONB DEFAULT '{}',
    _INDEX BIGINT DEFAULT 0,
		PRIMARY KEY(TABLE_SCHEMA, TABLE_NAME)
	);
  CREATE INDEX IF NOT EXISTS STRUCTS__INDEX_IDX ON linq.MODELS(_INDEX);
	CREATE INDEX IF NOT EXISTS STRUCTS_SCHEMA_IDX ON linq.MODELS(TABLE_SCHEMA);
	CREATE INDEX IF NOT EXISTS STRUCTS_NAME_IDX ON linq.MODELS(TABLE_NAME);

	CREATE OR REPLACE FUNCTION linq.setmodel(
	VSCHEMA VARCHAR(80),
	VNAME VARCHAR(80),
	VDEFINTION JSONB)
	RETURNS INT AS $$
	DECLARE
	 result INT;
	BEGIN
	 INSERT INTO linq.MODELS AS A (TABLE_SCHEMA, TABLE_NAME, DEFINTION)
	 SELECT VSCHEMA, VNAME, VDEFINTION
	 ON CONFLICT (TABLE_SCHEMA, TABLE_NAME) DO UPDATE SET
	 DEFINTION = VDEFINTION
	 RETURNING _INDEX INTO result;

	 RETURN COALESCE(result, 0);
	END;
	$$ LANGUAGE plpgsql;

	DROP TRIGGER IF EXISTS SERIES_INSERT ON linq.MODELS CASCADE;
	CREATE TRIGGER SERIES_INSERT
	BEFORE INSERT ON linq.MODELS
	FOR EACH ROW
	EXECUTE PROCEDURE linq.SERIES_INSERT();

	DROP TRIGGER IF EXISTS SERIES_UPDATE ON linq.MODELS CASCADE;
	CREATE TRIGGER SERIES_UPDATE
	AFTER UPDATE ON linq.MODELS
	FOR EACH ROW WHEN (NEW!=OLD)
	EXECUTE PROCEDURE linq.SERIES_UPDATE();
	`
}

// ddlFuntions return sql funcitions ddl to support a models
func ddlFuntions() string {
	return `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	CREATE SCHEMA IF NOT EXISTS linq;
	
	CREATE OR REPLACE FUNCTION linq.create_constraint_if_not_exists(
	s_name text,
	t_name text,
	c_name text,
	constraint_sql text) 
	RETURNS void AS $$
	BEGIN
		IF NOT EXISTS(
		SELECT constraint_name 
		FROM information_schema.table_constraints 
		WHERE UPPER(table_schema)=UPPER(s_name)
		AND UPPER(table_name)=UPPER(t_name)
		AND UPPER(constraint_name)=UPPER(c_name)) THEN
		 execute constraint_sql;
		END IF;
	END;
	$$ LANGUAGE 'plpgsql';
	`
}

// ddlDedault return sql default values
func ddlDefault(col *linq.Column) string {
	var result string
	switch col.TypeData {
	case linq.TpKey:
		result = `'-1'`
	case linq.TpText:
		result = `''`
	case linq.TpMemo:
		result = `''`
	case linq.TpNumber:
		result = `0`
	case linq.TpDate:
		result = `NOW()`
	case linq.TpCheckbox:
		result = `FALSE`
	case linq.TpRelation:
		result = `''`
	case linq.TpRollup:
		result = `''`
	case linq.TpCreatedTime:
		result = `NOW()`
	case linq.TpCreatedBy:
		result = `'{ "_id": "", "name": "" }'`
	case linq.TpLastEditedTime:
		result = `NOW()`
	case linq.TpLastEditedBy:
		result = `'{ "_id": "", "name": "" }'`
	case linq.TpStatus:
		result = `'{ "_id": "0", "main": "State", "name": "Activo" }'`
	case linq.TpPerson:
		result = `'{ "_id": "", "name": "" }'`
	case linq.TpFile:
		result = `''`
	case linq.TpURL:
		result = `''`
	case linq.TpEmail:
		result = `''`
	case linq.TpPhone:
		result = `''`
	case linq.TpFormula:
		result = `''`
	case linq.TpSelect:
		result = `''`
	case linq.TpMultiSelect:
		result = `''`
	case linq.TpJson:
		result = `'{}'`
	case linq.TpArray:
		result = `'[]'`
	case linq.TpSerie:
		result = `0`
	default:
		val := col.Default
		result = strs.Format(`%v`, et.Quote(val))
	}

	return strs.Append("DEFAULT", result, " ")
}

// ddlType return sql type ddl
func ddlType(col *linq.Column) string {
	switch col.TypeData {
	case linq.TpKey, linq.TpRelation, linq.TpRollup, linq.TpStatus, linq.TpPhone, linq.TpSelect, linq.TpMultiSelect:
		return "VARCHAR(80)"
	case linq.TpMemo:
		return "TEXT"
	case linq.TpNumber:
		return "DECIMAL(18, 2)"
	case linq.TpDate:
		return "TIMESTAMP"
	case linq.TpCheckbox:
		return "BOOLEAN"
	case linq.TpCreatedTime:
		return "TIMESTAMP"
	case linq.TpCreatedBy:
		return "JSONB"
	case linq.TpLastEditedTime:
		return "TIMESTAMP"
	case linq.TpLastEditedBy:
		return "JSONB"
	case linq.TpPerson:
		return "JSONB"
	case linq.TpFile:
		return "JSONB"
	case linq.TpURL:
		return "TEXT"
	case linq.TpFormula:
		return "JSONB"
	case linq.TpJson:
		return "JSONB"
	case linq.TpArray:
		return "JSONB"
	case linq.TpSerie:
		return "BIGINT"
	default:
		return "VARCHAR(250)"
	}
}

// ddlschema return sql schema ddl
func ddlSchema(schema *linq.Schema) string {
	return strs.Format(`CREATE SCHEMA IF NOT EXISTS %s;`, schema.Name)
}

// ddlColumn return sql column ddl
func ddlColumn(col *linq.Column) string {
	var result string
	var def string

	result = ddlDefault(col)
	def = ddlType(col)
	result = strs.Append(def, result, " ")
	result = strs.Append(col.Up(), result, " ")

	return result
}

// ddlIndex return sql index ddl
func ddlIndex(col *linq.Column) string {
	name := strs.Format(`%v_%v_IDX`, strs.Uppcase(col.Table()), col.Up())
	name = strs.Replace(name, "-", "_")
	name = strs.Replace(name, ".", "_")
	return strs.Format(`CREATE INDEX IF NOT EXISTS %v ON %v(%v);`, name, strs.Uppcase(col.Table()), col.Up())
}

// ddlUnique return sql unique index ddl
func ddlUnique(col *linq.Column) string {
	name := strs.Format(`%v_%v_IDX`, strs.Uppcase(col.Table()), col.Up())
	name = strs.Replace(name, "-", "_")
	name = strs.Replace(name, ".", "_")
	return strs.Format(`CREATE UNIQUE INDEX IF NOT EXISTS %v ON %v(%v);`, name, strs.Uppcase(col.Table()), col.Up())
}

// ddlPrimaryKey return sql primary key ddl
func ddlPrimaryKey(col *linq.Column) string {
	schema := strs.Uppcase(col.Model.Schema.Name)
	key := strs.Replace(col.Table(), ".", "_")
	key = strs.Replace(key, "-", "_") + "_pkey"
	key = strs.Lowcase(key)
	def := strs.Format(`ALTER TABLE IF EXISTS %s ADD CONSTRAINT %s PRIMARY KEY (%s);`, strs.Uppcase(col.Table()), key, strings.Join(col.PrimaryKeys(), ", "))
	return strs.Format(`SELECT linq.create_constraint_if_not_exists('%s', '%s', '%s', '%s');`, schema, col.Model.Name, key, def)
}

// ddlForeignKeys return ForeignKey ddl
func ddlForeignKeys(model *linq.Model) string {
	var result string
	for _, ref := range model.ForeignKey {
		schema := strs.Uppcase(model.Schema.Name)
		key := strs.Replace(model.Table, ".", "_") + "_" + strings.Join(ref.ForeignKey, "_")
		key = strs.Replace(key, "-", "_") + "_fkey"
		key = strs.Lowcase(key)
		def := strs.Format(`ALTER TABLE IF EXISTS %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s);`, strs.Uppcase(model.Table), key, strings.Join(ref.ForeignKey, ", "), ref.Parent.Table, strings.Join(ref.ParentKey, ", "))
		def = strs.Format(`SELECT linq.create_constraint_if_not_exists('%s', '%s', '%s', '%s');`, schema, model.Name, key, def)
		result = strs.Append(result, def, "\n")
	}

	return result
}

// ddlSetSync return sql set sync ddl
func ddlSetSync(model *linq.Model) string {
	result := linq.SQLDDL(`
	DROP TRIGGER IF EXISTS SYNC_INSERT ON $1 CASCADE;
	CREATE TRIGGER SYNC_INSERT
	BEFORE INSERT ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE linq.SYNC_INSERT();

	DROP TRIGGER IF EXISTS SYNC_UPDATE ON $1 CASCADE;
	CREATE TRIGGER SYNC_UPDATE
	BEFORE UPDATE ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE linq.SYNC_UPDATE();

	DROP TRIGGER IF EXISTS SYNC_DELETE ON $1 CASCADE;
	CREATE TRIGGER SYNC_DELETE
	BEFORE DELETE ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE linq.SYNC_DELETE();`, strs.Uppcase(model.Table))

	result = strs.Replace(result, "\t", "")

	return result
}

func ddlSetRecycling(model *linq.Model) string {
	result := linq.SQLDDL(`	
	DROP TRIGGER IF EXISTS RECYCLING ON $1 CASCADE;
	CREATE TRIGGER RECYCLING
	AFTER UPDATE ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE linq.RECYCLING();

	DROP TRIGGER IF EXISTS ERASE ON $1 CASCADE;
	CREATE TRIGGER ERASE
	AFTER DELETE ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE linq.ERASE();`, strs.Uppcase(model.Table))

	result = strs.Replace(result, "\t", "")

	return result
}

func ddlSetSeries(model *linq.Model) string {
	result := linq.SQLDDL(`	
	DROP TRIGGER IF EXISTS SERIES_INSERT ON $1 CASCADE;
	CREATE TRIGGER SERIES_INSERT
	BEFORE INSERT ON $1
	FOR EACH ROW
	EXECUTE PROCEDURE linq.SERIES_INSERT();

	DROP TRIGGER IF EXISTS SERIES_UPDATE ON $1 CASCADE;
	CREATE TRIGGER SERIES_UPDATE
	AFTER UPDATE ON $1
	FOR EACH ROW WHEN (NEW!=OLD)
	EXECUTE PROCEDURE linq.SERIES_UPDATE();`, strs.Uppcase(model.Table))

	result = strs.Replace(result, "\t", "")

	return result
}

func ddlSetModel(model *linq.Model) string {
	schema := model.Schema.Name
	table := model.Name
	definition := model.Definition().ToString()
	result := linq.SQLDDL(`	
	SELECT linq.setmodel('$1', '$2', '$3');`, schema, table, definition)

	result = strs.Replace(result, "\t", "")

	return result
}

// ddlTable return table ddl
func ddlTable(model *linq.Model) string {
	var result string
	var columns string
	var indexs string
	for _, col := range model.Columns {
		if col.TypeColumn == linq.TpColumn {
			def := ddlColumn(col)
			columns = strs.Append(def, columns, ",\n")
			if col.PrimaryKey {
				def = ddlPrimaryKey(col)
				indexs = strs.Append(def, indexs, "\n")
			} else if col.Unique {
				def = ddlUnique(col)
				indexs = strs.Append(def, indexs, "\n")
			} else if col.Indexed {
				def = ddlIndex(col)
				indexs = strs.Append(def, indexs, "\n")
			}
		}
	}
	schema := ddlSchema(model.Schema)
	result = strs.Append(result, schema, "\n")
	table := strs.Format("CREATE TABLE IF NOT EXISTS %s (\n%s);", model.Table, columns)
	result = strs.Append(result, table, "\n")
	result = strs.Append(result, indexs, "\n\n")
	foreign := ddlForeignKeys(model)
	result = strs.Append(result, foreign, "\n\n")
	sync := ddlSetSync(model)
	result = strs.Append(result, sync, "\n\n")
	recycle := ddlSetRecycling(model)
	result = strs.Append(result, recycle, "\n\n")
	series := ddlSetSeries(model)
	result = strs.Append(result, series, "\n\n")
	model.DDL = result
	define := ddlSetModel(model)
	result = strs.Append(result, define, "\n\n")

	return result
}
