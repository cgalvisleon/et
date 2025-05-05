package jrpc

import (
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"reflect"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

type Package struct {
	Name     string    `json:"name"`
	Host     string    `json:"host"`
	Port     int       `json:"port"`
	Solvers  []*Solver `json:"routes"`
	Replicas int       `json:"replicas"`
	started  bool      `json:"-"`
}

/**
* getPackages
* @return []*Package, error
**/
func getPackages() ([]*Package, error) {
	var result []*Package
	bt, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	str, err := cache.Get(sourceKey, string(bt))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(str), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* serialize
* @return et.Json
**/
func (s *Package) serialize() et.Json {
	solvers := []et.Json{}
	for _, solver := range s.Solvers {
		solvers = append(solvers, solver.serialize())
	}

	return et.Json{
		"name":     s.Name,
		"host":     s.Host,
		"port":     s.Port,
		"solvers":  solvers,
		"replicas": s.Replicas,
	}
}

/**
* Describe
* @return et.Json
**/
func (s *Package) Describe() et.Json {
	return s.serialize()
}

/**
* Save
* @return error
**/
func (s *Package) save() error {
	packages, err := getPackages()
	if err != nil {
		return err
	}

	idx := slices.IndexFunc(packages, func(p *Package) bool { return p.Name == s.Name })
	if idx != -1 {
		s.Replicas = packages[idx].Replicas + 1
		packages[idx] = s
	} else {
		s.Replicas = 1
		packages = append(packages, s)
	}

	bt, err := json.Marshal(packages)
	if err != nil {
		return err
	}

	cache.Set(sourceKey, string(bt), 0)

	return nil
}

/**
* Start
**/
func (s *Package) start() error {
	if s.started {
		return nil
	}

	address := strs.Format(`:%d`, s.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logs.Fatal(err)
	}

	s.started = true
	logs.Logf("Rpc", `Running on %s%s`, s.Host, listener.Addr())

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
			PackageName: s.Name,
			Host:        s.Host,
			Port:        s.Port,
			StructName:  structName,
			Method:      name,
			Inputs:      inputs,
			Output:      outputs,
		}
		s.Solvers = append(s.Solvers, solver)
	}

	rpc.Register(services)

	return s.save()
}

/**
* UnMount
* @return error
**/
func (s *Package) unMount() error {
	packages, err := getPackages()
	if err != nil {
		return logs.Alert(err)
	}

	idx := slices.IndexFunc(packages, func(p *Package) bool { return p.Name == s.Name })
	if idx != -1 {
		pkg := packages[idx]
		pkg.Replicas = pkg.Replicas - 1
		if pkg.Replicas == 0 {
			packages = slices.Delete(packages, idx, idx+1)
		} else {
			packages[idx] = pkg
		}
	}

	bt, err := json.Marshal(packages)
	if err != nil {
		return err
	}

	cache.Set(sourceKey, string(bt), 0)

	return nil
}
