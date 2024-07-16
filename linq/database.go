package linq

import (
	"database/sql"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
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
	debug       bool
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
		return nil, logs.Alert(err)
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

	driver.SetListen(result.handlerListened)
	result.Driver = &driver

	dbs = append(dbs, result)

	return result, nil
}

/**
* beforeInsert set the values before insert
* @param model *Model
* @param value *Values
* @return error
**/
func beforeInsert(model *Model, value *Values) error {
	now := utility.Now()
	if model.ColumnCreatedTime != nil {
		value.Set(model.ColumnCreatedTime, now)
	}
	if model.ColumnLastEditedTime != nil {
		value.Set(model.ColumnLastEditedTime, now)
	}
	if model.ColumnSerie != nil {
		index, err := model.DB.NextSerie(model.Table)
		if err != nil {
			return logs.Alert(err)
		}

		value.Set(model.ColumnSerie, index)
	}

	return nil
}

/**
* beforeUpdate set the values before update
* @param model *Model
* @param value *Values
* @return error
**/
func beforeUpdate(model *Model, value *Values) error {
	now := utility.Now()
	if model.ColumnLastEditedTime != nil {
		value.Set(model.ColumnLastEditedTime, now)
	}

	return nil
}

/**
* handlerListened handler listened
* @param res et.Json
**/
func (d *Database) handlerListened(res et.Json) {
	logs.Debug("Lintened: ", res.ToString())
}

/**
* Describe return the definition of the database
* @return et.Json
**/
func (d *Database) Describe() et.Json {
	var _schemes []et.Json = []et.Json{}
	for _, s := range d.Schemes {
		_schemes = append(_schemes, s.Describe())
	}

	var _models []et.Json = []et.Json{}
	for _, m := range d.Models {
		_models = append(_models, m.Describe())
	}

	driver := *d.Driver
	typeDriver := driver.Type()

	return et.Json{
		"name":        d.Name,
		"description": d.Description,
		"typeDriver":  typeDriver,
		"sourceField": SourceField,
		"schemes":     _schemes,
		"models":      _models,
	}
}

/**
* Debug set debug mode
**/
func (d *Database) Debug() {
	d.debug = true
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
	for _, v := range d.Models {
		if strs.Uppcase(v.Name) == strs.Uppcase(name) {
			return v
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
	kind := "model"

	result, err := driver.GetModel(model.Schema.Name, model.Name, kind)
	if err != nil {
		return err
	}

	if !result.Ok {
		newColumn(model, IdTField.Low(), "", TpColumn, TpKey, TpKey.Default())
		sql := driver.DefineSql(model)
		if d.debug {
			logs.Debug(model.Describe().ToString())
		}

		_, err = Exec(d.DB, sql)
		if err != nil {
			return err
		}

		err = driver.InsertModel(model.Schema.Name, model.Name, kind, model.Version, model.Describe())
		if err != nil {
			return err
		}

		return nil
	}

	version := result.Int("version")
	if version < model.Version {
		sql := driver.MutationSql(model)
		if d.debug {
			logs.Debug(model.Describe().ToString())
		}

		_, err = Exec(d.DB, sql)
		if err != nil {
			return err
		}

		err = driver.UpdateModel(model.Schema.Name, model.Name, kind, model.Version, model.Describe())
		if err != nil {
			return err
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
* @return et.Items
* @return error
**/
func (d *Database) Query(sql string, args ...any) (et.Items, error) {
	_query := SQLParse(sql, args...)

	if d.debug {
		logs.Debug(_query)
	}

	items, err := Query(d.DB, _query)
	if err != nil {
		return et.Items{}, err
	}

	return items, nil
}

/**
* QueryOne execute a query in the database and return one item
* @parms db
* @parms sql
* @parms args
* @return et.Item
* @return error
**/
func (d *Database) QueryOne(sql string, args ...any) (et.Item, error) {
	items, err := d.Query(sql, args...)
	if err != nil {
		return et.Item{}, err
	}

	if items.Count == 0 {
		return et.Item{
			Ok:     false,
			Result: et.Json{},
		}, nil
	}

	return et.Item{
		Ok:     items.Ok,
		Result: items.Result[0],
	}, nil
}

/**
* Exec execute a query
* @param sql string
* @param args ...any
* @return et.Items
* @return error
**/
func (d *Database) Data(source, sql string, args ...any) (et.Items, error) {
	rows, err := query(d.DB, sql, args...)
	if err != nil {
		return et.Items{}, logs.Error(err)
	}
	defer rows.Close()

	var result et.Items = et.Items{}
	for rows.Next() {
		var item et.Item
		err := item.Scan(rows)
		if err != nil {
			continue
		}

		result.Result = append(result.Result, item.Json(source))
		result.Ok = true
		result.Count++
	}

	return result, nil
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
