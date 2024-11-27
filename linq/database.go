package linq

import (
	"database/sql"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/rt"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/ws"
)

// Mode Database
type ModeDatabase int

/**
* String return string of mode database
* @return string
**/
func (m ModeDatabase) String() string {
	switch m {
	case ModeNone:
		return "None"
	case ModeCore:
		return "Core"
	case ModeMaster:
		return "Master"
	case ModeRead:
		return "Read"
	case ModeMocks:
		return "Mocks"
	}
	return ""
}

/**
* IsNone return if the mode is none
* @return bool
**/
func (m ModeDatabase) IsNone() bool {
	return m == ModeNone
}

/**
* ToModeDatabase convert int to ModeDatabase
* @param val int
* @return ModeDatabase
**/
func ToModeDatabase(val int) ModeDatabase {
	switch val {
	case 1:
		return ModeCore
	case 2:
		return ModeMaster
	default:
		return ModeNone
	}
}

const (
	ModeNone ModeDatabase = iota
	ModeCore
	ModeMaster
	ModeRead
	ModeMocks
)

/**
* DB struct to define a database
**/
type DB struct {
	Index       int
	Name        string
	Description string
	DB          *sql.DB
	WS          *ws.Hub
	Driver      *Driver
	Schemes     []*Schema
	Models      []*Model
	loadMaster  bool
}

/**
* NewDB create a new database
* @param name string
* @param description string
* @param drive Driver
* @return *DB
**/
func NewDB(name string, db *sql.DB, driver Driver) (*DB, error) {
	for _, v := range dbs {
		if v.Name == strs.Uppcase(name) {
			return v, nil
		}
	}

	result := &DB{
		Index:   len(dbs) + 1,
		Name:    strs.Lowcase(name),
		DB:      db,
		Driver:  &driver,
		Schemes: []*Schema{},
		Models:  []*Model{},
	}

	switch driver.Mode() {
	case ModeMaster:
		result.WS = ws.NewHub()

		driver.SetListen([]string{"command"}, result.commandPublish)
		driver.SetListen([]string{"sync"}, result.syncListener)
		driver.SetListen([]string{"recycling"}, result.recyclingListener)
	case ModeCore:
		result.connectRT()
		driver.SetListen([]string{"command"}, result.commandPublish)
		driver.SetListen([]string{"sync"}, result.syncListener)
		driver.SetListen([]string{"recycling"}, result.recyclingListener)
	case ModeRead:
		result.connectRT()
		driver.SetListen([]string{"sync"}, result.syncListener)
		driver.SetListen([]string{"recycling"}, result.recyclingListener)
	case ModeMocks:
		result.connectRT()
		driver.SetListen([]string{"sync"}, result.syncListener)
		driver.SetListen([]string{"recycling"}, result.recyclingListener)
	}

	dbs = append(dbs, result)

	return result, nil
}

/**
* connectRT load the websocket
* @return error
**/
func (d *DB) connectRT() error {
	err := rt.Load()
	if err != nil {
		return logs.Panic(err)
	}

	driver := *d.Driver
	switch driver.Mode() {
	case ModeMaster:
		rt.Subscribe("command", d.wsListener)
	case ModeCore:
		rt.Subscribe("command", d.wsListener)
	case ModeRead:
		rt.Subscribe("command", d.wsListener)
	case ModeMocks:
		rt.Subscribe("command", d.wsMocksListener)
	}

	return nil
}

/**
* commandPublish handler listened
* @param res et.Json
**/
func (d *DB) commandPublish(res et.Json) {
	id := res.Str("_id")
	driver := *d.Driver
	item, err := driver.GetCommand(id)
	if logs.Alert(err) != nil {
		return
	}

	if !item.Ok {
		return
	}

	rt.Publish("command", item.Result)
}

/**
* syncListener handler listened
* @param res et.Json
**/
func (d *DB) syncListener(res et.Json) {
	schema := res.Str("schema")
	table := res.Str("table")
	model := d.Table(schema, table)
	if model != nil && model.OnListener != nil {
		model.OnListener(res)
	}
}

/**
* recyclingListener handler listened
* @param res et.Json
**/
func (d *DB) recyclingListener(res et.Json) {
	schema := res.Str("schema")
	table := res.Str("table")
	model := d.Table(schema, table)
	if model != nil && model.OnListener != nil {
		model.OnListener(res)
	}
}

/**
* wsListener handler listened
* @param msg message.Message
**/
func (d *DB) wsListener(msg ws.Message) {
	data, err := et.Object(msg.Data)
	if logs.Alert(err) != nil {
		return
	}

	id := data.Str("_id")
	query := data.Str("sql")
	index := data.Int64("index")
	driver := *d.Driver
	err = driver.SetMutex(id, query, index)
	if logs.Alert(err) != nil {
		return
	}

	logs.Debug(msg)
}

/**
* wsMocksListener handler listened
* @param msg message.Message
**/
func (d *DB) wsMocksListener(msg ws.Message) {
	data, err := et.Object(msg.Data)
	if logs.Alert(err) != nil {
		return
	}

	id := data.Str("_id")
	query := data.Str("sql")
	index := data.Int64("index")
	driver := *d.Driver
	err = driver.SetMutex(id, query, index)
	if logs.Alert(err) != nil {
		return
	}

	logs.Debug(msg)
}

/**
* Describe return the definition of the database
* @return et.Json
**/
func (d *DB) Describe() et.Json {
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
* Close the database
**/
func (d *DB) Close() {
	if d.DB != nil {
		d.DB.Close()
	}
	if d.WS != nil {
		d.WS.Close()
	}
}

/**
* Master set the database as master
* @param params *Connection
* @return error
**/
func (d *DB) Master(params *Connection) error {
	if d.loadMaster {
		return nil
	}

	driver := *d.Driver
	_, err := driver.Master(params)
	if err != nil {
		return err
	}

	d.loadMaster = true

	return nil
}

/**
* Read set the database as master
* @param params *Connection
* @return error
**/
func (d *DB) Read(params *Connection) error {
	if d.loadMaster {
		return nil
	}

	driver := *d.Driver
	_, err := driver.Master(params)
	if err != nil {
		return err
	}

	return nil
}

/**
* InitSchema init a schema
* @param schema *Schema
* @return error
**/
func (d *DB) InitModel(model *Model) error {
	if d.DB == nil {
		return logs.Alertm(MSG_CONNECTED_REQUIRED)
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
func (d *DB) GetSchema(schema *Schema) *Schema {
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
func (d *DB) GetModel(model *Model) *Model {
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
func (d *DB) Model(name string) *Model {
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
func (d *DB) Table(schema, name string) *Model {
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
func (d *DB) Disconnected() error {
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
func (d *DB) initModel(model *Model) error {
	if d.Driver == nil {
		return logs.Alertm(MSG_DRIVER_REQUIRED)
	}

	driver := *d.Driver
	if driver.Mode().IsNone() {
		return nil
	}

	model.DefineTrigger(BeforeInsert, beforeInsert)
	model.DefineTrigger(AfterInsert, afterInsert)
	model.DefineTrigger(BeforeUpdate, beforeUpdate)
	model.DefineTrigger(AfterUpdate, afterUpdate)
	model.DefineTrigger(BeforeDelete, beforeDelete)
	model.DefineTrigger(AfterDelete, afterDelete)

	exist, err := driver.ModelExist(model.Schema.Name, model.Name)
	if err != nil {
		return err
	}

	if !exist {
		sql := driver.DefineSql(model)
		err = d.Exec(sql)
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
func (d *DB) selectSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm(MSG_DRIVER_REQUIRED)
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
func (d *DB) currentSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm(MSG_DRIVER_REQUIRED)
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
func (d *DB) insertSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm(MSG_DRIVER_REQUIRED)
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
func (d *DB) updateSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm(MSG_DRIVER_REQUIRED)
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
func (d *DB) deleteSql(linq *Linq) (string, error) {
	if d.Driver == nil {
		return "", logs.Errorm(MSG_DRIVER_REQUIRED)
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
func (d *DB) Query(sql string, args ...any) (et.Items, error) {
	rows, err := query(d, sql, args...)
	if err != nil {
		return et.Items{}, err
	}
	defer rows.Close()

	result := RowsToItems(rows)

	return result, nil
}

/**
* QueryOne execute a query in the database and return one item
* @parms db
* @parms sql
* @parms args
* @return et.Item
* @return error
**/
func (d *DB) QueryOne(sql string, args ...any) (et.Item, error) {
	items, err := d.Query(sql, args...)
	if err != nil {
		return et.Item{}, err
	}

	if items.Count == 0 {
		return et.Item{}, nil
	}

	result := et.Item{
		Ok:     true,
		Result: items.Result[0],
	}

	return result, nil
}

/**
* Command execute a query in the database and return one item
* @parms db
* @parms sql
* @parms args
* @return et.Item
* @return error
**/
func (d *DB) Command(sql string, args ...any) (et.Item, error) {
	rows, err := command(d, sql, args...)
	if err != nil {
		return et.Item{}, err
	}
	defer rows.Close()

	result := RowsToItem(rows)

	return result, nil
}

/**
* Exec execute a query
* @param sql string
* @param args ...any
* @return error
**/
func (d *DB) Exec(sql string, args ...any) error {
	_, err := d.Command(sql, args...)
	if err != nil {
		return err
	}

	return nil
}

/**
* Data execute a query
* @param sql string
* @param args ...any
* @return et.Items
* @return error
**/
func (d *DB) Data(source, sql string, args ...any) (et.Items, error) {
	rows, err := query(d, sql, args...)
	if err != nil {
		return et.Items{}, err
	}
	defer rows.Close()

	result := DataToItems(rows, SourceField.Low())

	return result, nil
}

/**
* DataOne execute a query and return one item
* @param sql string
* @param args ...any
* @return et.Item
* @return error
**/
func (d *DB) DataOne(source, sql string, args ...any) (et.Item, error) {
	rows, err := query(d, sql, args...)
	if err != nil {
		return et.Item{}, err
	}
	defer rows.Close()

	result := DataToItem(rows, SourceField.Low())

	return result, nil
}

/**
* NextSerie return the next serie
* @param tag string
* @return int
* @return error
**/
func (d *DB) NextSerie(tag string) (int, error) {
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
func (d *DB) NextCode(tag, format string) (string, error) {
	driver := *d.Driver
	return driver.NextCode(tag, format)
}

/**
* SetSerie set the serie
* @param tag string
* @param val int
* @return error
**/
func (d *DB) SetSerie(tag string, val int) error {
	driver := *d.Driver
	return driver.SetSerie(tag, val)
}

/**
* CurrentSerie return the current serie
* @param tag string
* @return int
* @return error
**/
func (d *DB) CurrentSerie(tag string) (int, error) {
	driver := *d.Driver
	return driver.CurrentSerie(tag)
}

/**
* DeleteSerie delete the serie
* @param tag string
* @return error
**/
func (d *DB) DeleteSerie(tag string) error {
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
func (d *DB) GetMigrateId(old_id, tag string) (string, error) {
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
func (d *DB) UpSertMigrateId(old_id, _id, tag string) error {
	driver := *d.Driver
	return driver.UpSertMigrateId(old_id, _id, tag)
}
