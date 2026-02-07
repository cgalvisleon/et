package jrpc

import (
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

type Package struct {
	Name    string    `json:"name"`
	Host    string    `json:"host"`
	Port    int       `json:"port"`
	Solvers []*Solver `json:"routes"`
	started bool      `json:"-"`
}

/**
* NewPackage
* @param name string, host string, port int
* @return *Package
**/
func NewPackage(name string, host string, port int) *Package {
	return &Package{
		Name:    name,
		Host:    host,
		Port:    port,
		Solvers: make([]*Solver, 0),
	}
}

/**
* ToJson
* @return et.Json
**/
func (s *Package) ToJson() et.Json {
	dt, err := json.Marshal(s)
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(dt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* Start
**/
func (s *Package) start() error {
	if s.started {
		return nil
	}

	address := fmt.Sprintf(`:%d`, s.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logs.Fatal(err)
	}

	s.started = true
	logs.Logf("Rpc", `%s running on %s%s`, s.Name, s.Host, listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			logs.Panic(err)
			continue
		}

		go rpc.ServeConn(conn)
	}
}

/**
* Mount
* @param services any
* @return error
**/
func (s *Package) Mount(services any) error {
	solvers, err := Mount(s.Host, services)
	if err != nil {
		return err
	}

	for method, solver := range solvers {
		s.Solvers = append(s.Solvers, &Solver{
			Method: method,
			Inputs: solver.ArrayStr("inputs"),
			Output: solver.ArrayStr("output"),
		})
	}

	return nil
}
