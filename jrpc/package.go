package jrpc

import (
	"fmt"
	"net"
	"net/rpc"
	"reflect"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
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
* Describe
* @return et.Json
**/
func (s *Package) Describe() et.Json {
	solvers := []et.Json{}
	for _, solver := range s.Solvers {
		solvers = append(solvers, solver.serialize())
	}

	return et.Json{
		"name":    s.Name,
		"host":    s.Host,
		"port":    s.Port,
		"count":   len(s.Solvers),
		"solvers": solvers,
	}
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
func (s *Package) mount(services any) error {
	tipoStruct := reflect.TypeOf(services)
	structName := tipoStruct.String()
	list := strings.Split(structName, ".")
	structName = list[len(list)-1]
	for i := 0; i < tipoStruct.NumMethod(); i++ {
		metodo := tipoStruct.Method(i)
		numInputs := metodo.Type.NumIn()
		numOutputs := metodo.Type.NumOut()

		inputs := et.Json{}
		for i := 1; i < numInputs; i++ {
			name := fmt.Sprintf(`param_%d`, i)
			paramType := metodo.Type.In(i)
			inputs[name] = paramType.String()
		}

		outputs := []string{}
		for i := 0; i < numOutputs; i++ {
			paramType := metodo.Type.Out(i)
			outputs = append(outputs, paramType.String())
		}

		structName = strs.DaskSpace(structName)
		name := strs.DaskSpace(metodo.Name)
		solver := &Solver{
			Host:       s.Host,
			Port:       s.Port,
			StructName: structName,
			Method:     name,
			Inputs:     inputs,
			Output:     outputs,
		}
		s.Solvers = append(s.Solvers, solver)
	}

	rpc.Register(services)

	return s.save()
}
