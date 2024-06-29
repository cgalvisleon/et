package lib

import (
	"database/sql"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	_ "github.com/mattn/go-sqlite3"
)

// Sqlite struct to define a sqlite database
type Sqlite struct {
	Database string
	DB       *sql.DB
}

func Load(database string) Sqlite {
	return Sqlite{
		Database: database,
	}
}

// Type return the type of the driver
func (d *Sqlite) Type() string {
	return linq.Sqlite.String()
}

// Connect to the database
func (d *Sqlite) Connect(params et.Json) (*sql.DB, error) {
	driver := "sqlite3"

	result, err := sql.Open(driver, d.Database)
	if err != nil {
		return nil, err
	}

	err = result.Ping()
	if err != nil {
		return nil, err
	}

	d.DB = result

	logs.Infof("Connected to %s database %s", driver, d.Database)

	return d.DB, nil
}

// DDLModel return the ddl to create the model
func (d *Sqlite) DdlSql(m *linq.Model) string {
	var result string

	result = ddlTable(m)

	return result
}

// SelectSql return the sql to select
func (d *Sqlite) SelectSql(l *linq.Linq) string {
	sqlSelect(l)

	sqlFrom(l)

	sqlJoin(l)

	sqlWhere(l)

	sqlGroupBy(l)

	sqlHaving(l)

	sqlOrderBy(l)

	sqlLimit(l)

	sqlOffset(l)

	return l.Sql
}

// CurrentSql return the sql to get the current
func (d *Sqlite) CurrentSql(l *linq.Linq) string {
	sqlCurrent(l)

	sqlFrom(l)

	sqlWhere(l)

	sqlLimit(l)

	return l.Sql
}

// InsertSql return the sql to insert
func (d *Sqlite) InsertSql(l *linq.Linq) string {
	sqlInsert(l)

	sqlReturns(l)

	return l.Sql
}

// UpdateSql return the sql to update
func (d *Sqlite) UpdateSql(l *linq.Linq) string {
	sqlUpdate(l)

	sqlWhere(l)

	sqlReturns(l)

	return l.Sql
}

// DeleteSql return the sql to delete
func (d *Sqlite) DeleteSql(l *linq.Linq) string {
	sqlDelete(l)

	sqlReturns(l)

	return l.Sql
}

// DCL Data Control Language execute a command
func (d *Sqlite) DCL(command string, params et.Json) error {
	return nil
}

// MutationSql return the sql to mutate tables
func (d *Sqlite) MutationSql(l *linq.Linq) string {

	return ""
}
