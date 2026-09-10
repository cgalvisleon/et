package graph

import (
	"context"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const PackageName = "graph"

var conn *Conn

type Conn struct {
	driver neo4j.DriverWithContext
	id     string
	host   string
}

/**
* Load connects to Neo4j using NEO4J_HOST, NEO4J_USER and NEO4J_PASSWORD.
* @return (*Conn, error)
**/
func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	host := envar.GetStr("NEO4J_HOST", "")
	if !utility.ValidStr(host, 0, []string{}) {
		return nil, logs.Alertf(msg.MSG_ATRIB_REQUIRED, "host")
	}

	user := envar.GetStr("NEO4J_USER", "")
	password := envar.GetStr("NEO4J_PASSWORD", "")

	driver, err := neo4j.NewDriverWithContext(host, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		return nil, err
	}

	if err := driver.VerifyConnectivity(context.Background()); err != nil {
		return nil, err
	}

	logs.Logf(PackageName, "Connected host:%s", host)

	conn = &Conn{
		driver: driver,
		id:     utility.UUID(),
		host:   host,
	}

	return conn, nil
}

/**
* Close closes the underlying Neo4j driver connection.
* @return error
**/
func (s *Conn) Close() error {
	return s.driver.Close(context.Background())
}

/**
* HealthCheck reports whether the driver can still reach the server.
* @return bool
**/
func (s *Conn) HealthCheck() bool {
	return s.driver.VerifyConnectivity(context.Background()) == nil
}
