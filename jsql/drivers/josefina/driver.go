package josefina

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/jsql/drivers/postgres"
	_ "github.com/lib/pq"
)

/**
* Josefina: Driver de jsql para Josefina (B5). Josefina habla el protocolo de PostgreSQL
* (PG_PORT) y entiende SQL con su lenguaje jql, que una conexión elige con
* options=-c language=jql: el SQL lo genera el driver de Postgres y la conexión es la de
* lib/pq. Josefina no crea ni borra bases por SQL (se crean con su API): Connect y
* CreateDB solo comprueban que la base exista, y DropDB no está soportado.
**/
type Josefina struct {
	postgres.Postgres
}

func init() {
	jsql.Register(jsql.DriverJosefina, &Josefina{})
}

/**
* chain: La cadena de conexión a la base de Josefina con el lenguaje jql
* @param params et.Json
* @return string
**/
func chain(params et.Json) string {
	host := params.ValStr("localhost", "host")
	port := params.ValInt(5432, "port")
	user := params.ValStr("admin", "user")
	password := params.ValStr("", "password")
	database := params.ValStr("josefina", "database")
	sslMode := params.ValStr("disable", "sslmode")
	if sslMode == "" {
		sslMode = "disable"
	}
	appName := params.ValStr("et", "app_name")
	query := url.Values{}
	query.Set("sslmode", sslMode)
	query.Set("application_name", appName)
	query.Set("options", "-c language=jql")
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?%s",
		url.QueryEscape(user), url.QueryEscape(password), host, port, url.PathEscape(database), query.Encode())
}

/**
* connectTo: Abre la conexión y comprueba que responde
* @param ctx context.Context, dsn string
* @return *sql.DB, error
**/
func connectTo(ctx context.Context, dsn string) (*sql.DB, error) {
	result, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := result.PingContext(ctx); err != nil {
		result.Close()
		return nil, err
	}
	return result, nil
}

/**
* Connect: Se conecta a la base de Josefina de db.Params (que tiene que existir). Sin
* timeout, o con 0, no vence.
* @param db *jsql.DB, timeout ...time.Duration
* @return *sql.DB, error
**/
func (s *Josefina) Connect(db *jsql.DB, timeout ...time.Duration) (*sql.DB, error) {
	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()

	params := db.Params
	result, err := connectTo(ctx, chain(params))
	if err != nil {
		return nil, err
	}
	result.SetMaxOpenConns(params.ValInt(3, "pool_max_open"))
	result.SetMaxIdleConns(params.ValInt(1, "pool_max_idle"))
	result.SetConnMaxLifetime(time.Duration(params.ValInt(30, "pool_lifetime")) * time.Minute)
	result.SetConnMaxIdleTime(time.Duration(params.ValInt(2, "pool_idle_time")) * time.Minute)
	return result, nil
}

/**
* CreateDB: Comprueba que la base exista (Josefina las crea con su API, no por SQL)
* @param connection *jsql.ConnectParams, timeout ...time.Duration
* @return error
**/
func (s *Josefina) CreateDB(connection *jsql.ConnectParams, timeout ...time.Duration) error {
	if connection == nil || connection.Connection == nil {
		return fmt.Errorf(jsql.MSG_ATRIB_REQUIRED, "connection")
	}
	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()

	db, err := connectTo(ctx, chain(connection.Connection.GetParams()))
	if err != nil {
		return fmt.Errorf("josefina: the database must exist (create it with Josefina's API): %w", err)
	}
	return db.Close()
}

/**
* DropDB: Josefina no borra bases por SQL (MSG_DRIVER_CANNOT_DROP_DB, como Oracle)
* @param db *jsql.DB, timeout ...time.Duration
* @return error
**/
func (s *Josefina) DropDB(db *jsql.DB, timeout ...time.Duration) error {
	return fmt.Errorf(jsql.MSG_DRIVER_CANNOT_DROP_DB, jsql.DriverJosefina)
}

// ErrCannotDrop: Josefina no borra modelos (tablas) por SQL
var ErrCannotDrop = fmt.Errorf("josefina: models are not dropped through SQL (Josefina keeps them; empty them with DELETE ... WHERE true)")

/**
* Drop: Josefina no borra modelos por SQL: ErrCannotDrop
* @param model *jsql.Model, timeout ...time.Duration
* @return error
**/
func (s *Josefina) Drop(model *jsql.Model, timeout ...time.Duration) error {
	return ErrCannotDrop
}

/**
* Test: Prueba una conexión (jsql.Tester): abre la base y corre la consulta
* @param ctx context.Context, connection jsql.Connection, query string
* @return error
**/
func (s *Josefina) Test(ctx context.Context, connection jsql.Connection, query string) error {
	db, err := connectTo(ctx, chain(connection.GetParams()))
	if err != nil {
		return err
	}
	return jsql.Probe(ctx, db, query)
}
