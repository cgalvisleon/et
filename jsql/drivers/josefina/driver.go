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
* Josefina: jsql driver for Josefina (B5). Josefina speaks the PostgreSQL protocol (PG_PORT) and
* understands SQL through its jql language, chosen per connection with options=-c language=jql:
* the SQL comes from the Postgres driver and the connection is lib/pq's. Josefina does not create
* or drop databases through SQL (they are created with its API): Connect and CreateDB only check
* that the database exists, and DropDB is not supported.
**/
type Josefina struct {
	postgres.Postgres
}

func init() {
	jsql.Register(jsql.DriverJosefina, &Josefina{})
}

/**
* chain: The connection string to the Josefina database with the jql language
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
	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     fmt.Sprintf("%s:%d", host, port),
		Path:     "/" + database,
		RawQuery: query.Encode(),
	}
	return dsn.String()
}

/**
* connectTo: Opens the connection and checks that it answers
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
* Connect: Connects to the Josefina database of db.Params (it must exist). None or 0 timeout
* means no timeout.
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
* CreateDB: Checks that the database exists (Josefina creates them with its API, not through SQL)
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
* DropDB: Josefina does not drop databases through SQL (MSG_DRIVER_CANNOT_DROP_DB, like Oracle)
* @param db *jsql.DB, timeout ...time.Duration
* @return error
**/
func (s *Josefina) DropDB(db *jsql.DB, timeout ...time.Duration) error {
	return fmt.Errorf(jsql.MSG_DRIVER_CANNOT_DROP_DB, jsql.DriverJosefina)
}

// ErrCannotDrop: Josefina does not drop models (tables) through SQL
var ErrCannotDrop = fmt.Errorf("josefina: models are not dropped through SQL (Josefina keeps them; empty them with DELETE ... WHERE true)")

/**
* Drop: Josefina does not drop models through SQL: ErrCannotDrop
* @param model *jsql.Model, timeout ...time.Duration
* @return error
**/
func (s *Josefina) Drop(model *jsql.Model, timeout ...time.Duration) error {
	return ErrCannotDrop
}

/**
* Test: Tests a connection (jsql.Tester): opens the database and runs the query
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
