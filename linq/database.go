package linq

import (
	"database/sql"

	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

/**
* Database struct to define a database
**/
type Database struct {
	Index       int
	Name        string
	Description string
	DB          *sql.DB
	Driver      *Driver
	SourceField string
	Schemes     []*Schema
	Models      []*Model
}

/**
* NewDatabase create a new database
* @param name string
* @param description string
* @param drive Driver
* @return *Database
**/
func NewDatabase(name, description string, driver Driver) (*Database, error) {
	for _, v := range dbs {
		if v.Name == strs.Uppcase(name) {
			return v, nil
		}
	}

	db, err := driver.Connect()
	if err != nil {
		return nil, err
	}

	result := &Database{
		Index:       len(dbs) + 1,
		Name:        strs.Lowcase(name),
		Description: description,
		DB:          db,
		Driver:      &driver,
		Schemes:     []*Schema{},
		Models:      []*Model{},
	}

	if driver.UsedCore() {
		driver.SetListen(result.handlerListened)
	}

	result.Driver = &driver

	dbs = append(dbs, result)

	return result, nil
}

/**
* handlerListened handler listened
* @param res js.Json
**/
func (d *Database) handlerListened(res js.Json) {
	schema := res.Str("schema")
	table := res.Str("table")
	model := d.Table(schema, table)
	if model != nil && model.OnListener != nil {
		model.OnListener(res)
	}
}

/**
* Describe return the definition of the database
* @return js.Json
**/
func (d *Database) Describe() js.Json {
	var _schemes []js.Json = []js.Json{}
	for _, s := range d.Schemes {
		_schemes = append(_schemes, s.Describe())
	}

	var _models []js.Json = []js.Json{}
	for _, m := range d.Models {
		_models = append(_models, m.Describe())
	}

	driver := *d.Driver
	typeDriver := driver.Type()

	return js.Json{
		"name":        d.Name,
		"description": d.Description,
		"typeDriver":  typeDriver,
		"sourceField": SourceField,
		"schemes":     _schemes,
		"models":      _models,
	}
}

/**
* InitSchema init a schema
* @param schema *Schema
* @return error
**/
func (d *Database) InitModel(model *Model) error {
	if d.DB == nil {
		return logs.Alertm("Connected is required")
	}

	for _, v := range d.Models {
		if v == model {
			return nil
		}
	}

	err := d.initModel(model)
	if err != nil {
		return err
	}

	model.DefineTrigger(BeforeInsert, beforeInsert)
	model.DefineTrigger(BeforeUpdate, beforeUpdate)

	d.GetSchema(model.Schema)
	d.GetModel(model)

	return nil
}

/**
* GetSchema get a schema
* @param schema *Schema
* @return *Schema
**/
func (d *Database) GetSchema(schema *Schema) *Schema {
	for _, v := range d.Schemes {
		if v == schema {
			return v
		}
	}

	d.Schemes = append(d.Schemes, schema)
	schemas = append(schemas, schema)

	return schema
}

/**
* GetModel get model by model
* @param model *Model
* @return *Model
**/
func (d *Database) GetModel(model *Model) *Model {
	for _, v := range d.Models {
		if v == model {
			return v
		}
	}

	model.DB = d
	driver := *d.Driver
	if driver.Type() == "sqlite" {
		model.Table = model.Name
	}
	d.Models = append(d.Models, model)
	models = append(models, model)

	return model
}

/**
* Model get model by name
* @param name string
* @return *Model
**/
func (d *Database) Model(name string) *Model {
	for _, m := range d.Models {
		if strs.Uppcase(m.Name) == strs.Uppcase(name) {
			return m
		}
	}

	return nil
}

/**
* Table get model by tablename
* @param schema string
* @param name string
* @return *Model
**/
func (d *Database) Table(schema, name string) *Model {
	table := strs.Format("%s.%s", schema, name)
	for _, m := range d.Models {
		if strs.Uppcase(m.Table) == strs.Uppcase(table) {
			return m
		}
	}

	return nil
}

/**
* Disconnected to database
* @return error
**/
func (d *Database) Disconnected() error {
	if d.DB != nil {
		return d.DB.Close()
	}

	return nil
}

/**
* initModel init a model
* @param model *Model
* @return error
**/
func (d *Database) initModel(model *Model) error {
	if d.Driver == nil {
		return logs.Alertm("Driver is required")
	}

	driver := *d.Driver
	if !driver.UsedCore() {
		return nil
	}

	newColumn(model, IdTField.Low(), "Universal key", TpColumn, TpKey, TpKey.Default())

	exist, err := driver.ModelExist(model.Schema.Name, model.Name)
	if err != nil {
		return err
	}

	if !exist {
		sql := driver.DefineSql(model)
		_, err = Query(d.DB, sql)
		if err != nil {
			return logs.Alertf("%s\n%s", err.Error(), sql)
		}
	}

	return nil
}

/**
* selectSql return the sql to select
* @param linq *Linq
* @return string
* @return error
**/
func (d *Database) selectSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm("Driver is required")
	}

	driver := *d.Driver
	return driver.SelectSql(linq), nil
}

/**
* currentSql return the sql to current
* @param linq *Linq
* @return string
* @return error
**/
func (d *Database) currentSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm("Driver is required")
	}

	driver := *d.Driver
	return driver.CurrentSql(linq), nil
}

/**
* insertSql return the sql to insert
* @param linq *Linq
* @return string
* @return error
**/
func (d *Database) insertSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm("Driver is required")
	}

	driver := *d.Driver
	return driver.InsertSql(linq), nil
}

/**
* updateSql return the sql to update
* @param linq *Linq
* @return string
* @return error
**/
func (d *Database) updateSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm("Driver is required")
	}

	driver := *d.Driver
	return driver.UpdateSql(linq), nil
}

/**
* deleteSql return the sql to delete
* @param linq *Linq
* @return string
* @return error
**/
func (d *Database) deleteSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm("Driver is required")
	}

	driver := *d.Driver
	return driver.DeleteSql(linq), nil
}

/**
* Query execute a query in the database
* @parms db
* @parms sql
* @parms args
* @return js.Items
* @return error
**/
func (d *Database) Query(sql string, args ...any) (js.Items, error) {
	result, err := Query(d.DB, sql, args...)
	if err != nil {
		return js.Items{}, err
	}

	return result, nil
}

/**
* QueryOne execute a query in the database and return one item
* @parms db
* @parms sql
* @parms args
* @return js.Item
* @return error
**/
func (d *Database) QueryOne(sql string, args ...any) (js.Item, error) {
	result, err := QueryOne(d.DB, sql, args...)
	if err != nil {
		return js.Item{}, err
	}

	return result, nil
}

/**
* Exec execute a query
* @param sql string
* @param args ...any
* @return js.Items
* @return error
**/
func (d *Database) Exec(sql string, args ...any) (js.Item, error) {
	result, err := Exec(d.DB, sql, args...)
	if err != nil {
		return js.Item{}, err
	}

	return result, nil
}

/**
* Data execute a query
* @param sql string
* @param args ...any
* @return js.Items
* @return error
**/
func (d *Database) Data(source, sql string, args ...any) (js.Items, error) {
	result, err := Data(d.DB, sql, args...)
	if err != nil {
		return js.Items{}, err
	}

	return result, nil
}

/**
* DataOne execute a query and return one item
* @param sql string
* @param args ...any
* @return js.Item
* @return error
**/
func (d *Database) DataOne(source, sql string, args ...any) (js.Item, error) {
	result, err := DataOne(d.DB, sql, args...)
	if err != nil {
		return js.Item{}, err
	}

	return result, nil
}

/**
* UUIndex return the next index
* @param tag string
* @return int64
* @return error
**/
func (d *Database) UUIndex(tag string) (int64, error) {
	driver := *d.Driver
	return driver.UUIndex(tag)
}

/**
* NextSerie return the next serie
* @param tag string
* @return int
* @return error
**/
func (d *Database) NextSerie(tag string) (int, error) {
	driver := *d.Driver
	return driver.NextSerie(tag)
}

/**
* NextCode return the next code
* @param tag string
* @param format string
* @return string
* @return error
**/
func (d *Database) NextCode(tag, format string) (string, error) {
	driver := *d.Driver
	return driver.NextCode(tag, format)
}

/**
* SetSerie set the serie
* @param tag string
* @param val int
* @return error
**/
func (d *Database) SetSerie(tag string, val int) error {
	driver := *d.Driver
	return driver.SetSerie(tag, val)
}

/**
* CurrentSerie return the current serie
* @param tag string
* @return int
* @return error
**/
func (d *Database) CurrentSerie(tag string) (int, error) {
	driver := *d.Driver
	return driver.CurrentSerie(tag)
}

/**
* DeleteSerie delete the serie
* @param tag string
* @return error
**/
func (d *Database) DeleteSerie(tag string) error {
	driver := *d.Driver
	return driver.DeleteSerie(tag)
}

/**
* GetMigrateId get a migrate id
* @param old_id string
* @param tag string
* @return string
* @return error
**/
func (d *Database) GetMigrateId(old_id, tag string) (string, error) {
	driver := *d.Driver
	return driver.GetMigrateId(old_id, tag)
}

/**
* UpSertMigrateId upsert a migrate id
* @param old_id string
* @param _id string
* @param tag string
* @return error
**/
func (d *Database) UpSertMigrateId(old_id, _id, tag string) error {
	driver := *d.Driver
	return driver.UpSertMigrateId(old_id, _id, tag)
}
