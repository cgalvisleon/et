package linq

import (
	"database/sql"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

// Database struct used to define a database
type Database struct {
	Name        string
	Description string
	DB          *sql.DB
	Driver      *Driver
	SourceField string
	Schemes     []*Schema
	Models      []*Model
	debug       bool
}

// NewDatabase create a new database
func NewDatabase(name, description string, drive Driver) *Database {
	for _, v := range dbs {
		if v.Name == strs.Uppcase(name) {
			return v
		}
	}

	result := &Database{
		Name:        strs.Lowcase(name),
		Description: description,
		Driver:      &drive,
		Schemes:     []*Schema{},
		Models:      []*Model{},
	}

	dbs = append(dbs, result)

	return result
}

// Definition return a json with the definition of the database
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

func (d *Database) Debug() {
	d.debug = true
}

// AddModel add a model to the database
func (d *Database) InitModel(model *Model) error {
	if d.DB == nil {
		return logs.Errorm("Connected is required")
	}

	for _, v := range d.Models {
		if v == model {
			return nil
		}
	}

	model.SetDb(d)

	sql, err := d.ddlSql(model)
	if err != nil {
		return err
	}

	if d.debug {
		logs.Debug(model.Definition().ToString())
		logs.Debug(sql)
	}

	_, err = Exec(d.DB, sql)
	if err != nil {
		return err
	}

	model.ItIsBuilt = true

	return nil
}

// Get or add a schema to the database
func (d *Database) GetSchema(val *Schema) *Schema {
	for _, v := range d.Schemes {
		if v == val {
			return v
		}
	}

	d.Schemes = append(d.Schemes, val)
	schemas = append(schemas, val)

	return val
}

// Get or add a model to the database
func (d *Database) GetModel(val *Model) *Model {
	for _, v := range d.Models {
		if v == val {
			return v
		}
	}

	d.Models = append(d.Models, val)
	models = append(models, val)

	return val
}

func (d *Database) Model(name string) *Model {
	for _, v := range d.Models {
		if strs.Uppcase(v.Name) == strs.Uppcase(name) {
			return v
		}
	}

	return nil
}

// Connected to database
func (d *Database) Connected(params et.Json) error {
	if d.Driver == nil {
		return logs.Errorm("Driver is required")
	}

	var err error
	driver := *d.Driver
	d.DB, err = driver.Connect(params)
	if err != nil {
		return err
	}

	return nil
}

// Disconnected to database
func (d *Database) Disconnected() error {
	if d.DB != nil {
		return d.DB.Close()
	}

	return nil
}

// DDLModel return the ddl to create a model
func (d *Database) ddlSql(model *Model) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm("Driver is required")
	}

	driver := *d.Driver
	return driver.DdlSql(model), nil
}

// SelectSql return the sql to select
func (d *Database) selectSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm("Driver is required")
	}

	driver := *d.Driver
	return driver.SelectSql(linq), nil
}

// CurrentSql return the sql to current
func (d *Database) currentSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm("Driver is required")
	}

	driver := *d.Driver
	return driver.CurrentSql(linq), nil
}

// InsertSql return the sql to insert
func (d *Database) insertSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm("Driver is required")
	}

	driver := *d.Driver
	return driver.InsertSql(linq), nil
}

// UpdateSql return the sql to update
func (d *Database) updateSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm("Driver is required")
	}

	driver := *d.Driver
	return driver.UpdateSql(linq), nil
}

// DeleteSql return the sql to delete
func (d *Database) deleteSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm("Driver is required")
	}

	driver := *d.Driver
	return driver.DeleteSql(linq), nil
}
