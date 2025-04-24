package graph

import (
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const PackageName = "graph"

var (
	conn *Conn
)

type Conn struct {
	driver neo4j.DriverWithContext
	_id    string
	host   string
}

func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	driver, err := neo4j.NewDriverWithContext("neo4j://localhost:7687", neo4j.BasicAuth("neo4j", "password", ""))
	if err != nil {
		return nil, err
	}

	conn = &Conn{
		driver: driver,
		host:   "neo4j://localhost:7687",
	}

	return conn, nil
}
