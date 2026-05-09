package jrpc

import (
	"encoding/gob"
	"fmt"
	"net"
	"net/rpc"
	"reflect"
	"runtime"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

var (
	os   string
	rpcs map[string]et.Json
)

func init() {
	os = runtime.GOOS
	rpcs = make(map[string]et.Json)
	gob.Register(map[string]interface{}{})
	gob.Register(et.Json{})
	gob.Register(et.Item{})
	gob.Register(et.Items{})
	gob.Register(et.List{})
}

/**
* Mount
* @param host string, services any
* @return (*Package, error)
**/
func Mount(host string, port int, services any, packageName string) (*Package, error) {
	if pkg == nil {
		pkg = newPackage(packageName, host, port)
	}

	tipoStruct := reflect.TypeOf(services)
	structName := tipoStruct.String()
	list := strings.Split(structName, ".")
	structName = list[len(list)-1]
	for i := 0; i < tipoStruct.NumMethod(); i++ {
		metodo := tipoStruct.Method(i)
		numInputs := metodo.Type.NumIn()
		numOutputs := metodo.Type.NumOut()

		inputs := []string{}
		for i := 1; i < numInputs; i++ {
			paramType := metodo.Type.In(i)
			inputs = append(inputs, paramType.String())
		}

		outputs := []string{}
		for i := 0; i < numOutputs; i++ {
			paramType := metodo.Type.Out(i)
			outputs = append(outputs, paramType.String())
		}

		methodName := fmt.Sprintf("%s.%s", structName, metodo.Name)
		pkg.Add(methodName, inputs, outputs)
		logs.Logf("rpc", "RPC:/%s/%s", host, methodName)
	}

	err := rpc.Register(services)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

/**
* Start
* @param port int
**/
func Start(port int) error {
	address := fmt.Sprintf(`:%d`, port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				logs.Error(err)
				continue
			}

			go rpc.ServeConn(conn)
		}
	}()

	logs.Logf("Rpc", "running on %d", port)
	return nil
}

/**
* call: Calls a remote procedure
* @param host string, port int, method string, args any, reply any
* @return error
**/
func call(host string, port int, method string, args any, reply any) error {
	address := fmt.Sprintf("%s:%d", host, port)
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return ErrorRpcNotConnected
	}
	defer client.Close()

	err = client.Call(method, args, reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* Call
* @param method string, args any
* @return (any, error)
**/
func Call(method string, args any) (any, error) {
	solver, err := GetSolver(method)
	if err != nil {
		return nil, err
	}

	var reply any
	err = call(solver.Host, solver.Port, method, args, &reply)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

/**
* CallJson
* @param method string, args et.Json
* @return (et.Json, error)
**/
func CallJson(method string, args et.Json) (et.Json, error) {
	solver, err := GetSolver(method)
	if err != nil {
		return et.Json{}, err
	}

	var reply et.Json
	err = call(solver.Host, solver.Port, method, args, &reply)
	if err != nil {
		return et.Json{}, err
	}

	return reply, nil
}

/**
* CallItems
* @param method string, args et.Json
* @return (et.Items, error)
**/
func CallItems(method string, args et.Json) (et.Items, error) {
	solver, err := GetSolver(method)
	if err != nil {
		return et.Items{}, err
	}

	var reply et.Items
	err = call(solver.Host, solver.Port, method, args, &reply)
	if err != nil {
		return et.Items{}, err
	}

	return reply, nil
}

/**
* CallItem
* @param method string, args et.Json
* @return (et.Item, error)
**/
func CallItem(method string, args et.Json) (et.Item, error) {
	solver, err := GetSolver(method)
	if err != nil {
		return et.Item{}, err
	}

	var reply et.Item
	err = call(solver.Host, solver.Port, method, args, &reply)
	if err != nil {
		return et.Item{}, err
	}

	return reply, nil
}
