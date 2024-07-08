package linq

import (
	"database/sql"

	"github.com/cgalvisleon/et/et"
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
	debug       bool
}

/**
* NewDatabase create a new database
* @param name string
* @param description string
* @param drive Driver
* @return *Database
**/
func NewDatabase(name, description string, params et.Json, driver Driver) (*Database, error) {
	for _, v := range dbs {
		if v.Name == strs.Uppcase(name) {
			return v, nil
		}
	}

	db, err := driver.Connect(params)
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

	dbs = append(dbs, result)

	return result, nil
}

/**
* Definition return the definition of the database
* @return et.Json
**/
func (d *Database) Definition() et.Json {
	var _schemes []et.Json = []et.Json{}
	for _, s := range d.Schemes {
		_schemes = append(_schemes, s.Definition())
	}

	var _models []et.Json = []et.Json{}
	for _, m := range d.Models {
		_models = append(_models, m.Definition())
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

	model.Db = d
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
		newColumn(model, "ctid", "", TpPseudo, TpKey, "(0,0)")
		newColumn(model, IdTField.Low(), "", TpColumn, TpKey, TpKey.Default())
		sql := driver.DefineSql(model)
		if d.debug {
			logs.Debug(model.Definition().ToString())
			logs.Debug(sql)
		}

		_, err = Exec(d.DB, sql)
		if err != nil {
			return err
		}

		err = driver.InsertModel(model.Schema.Name, model.Name, kind, model.Version, model.Definition())
		if err != nil {
			return err
		}

		return nil
	}

	version := result.Int("version")
	if version < model.Version {
		sql := driver.MutationSql(model)
		if d.debug {
			logs.Debug(model.Definition().ToString())
			logs.Debug(sql)
		}

		_, err = Exec(d.DB, sql)
		if err != nil {
			return err
		}

		err = driver.UpdateModel(model.Schema.Name, model.Name, kind, model.Version, model.Definition())
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

func (d *Database) NextSerie(tag string) (int, error) {
	driver := *d.Driver
	return driver.NextSerie(tag)
}

func (d *Database) NextCode(tag, format string) (string, error) {
	driver := *d.Driver
	return driver.NextCode(tag, format)
}

func (d *Database) SetSerie(tag string, val int) error {
	driver := *d.Driver
	return driver.SetSerie(tag, val)
}

func (d *Database) CurrentSerie(tag string) (int, error) {
	driver := *d.Driver
	return driver.CurrentSerie(tag)
}

func (d *Database) DeleteSerie(tag string) error {
	driver := *d.Driver
	return driver.DeleteSerie(tag)
}
