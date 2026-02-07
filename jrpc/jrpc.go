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
* @return (map[string]et.Json, error)
**/
func Mount(host string, services any) (map[string]et.Json, error) {
	result := make(map[string]et.Json)
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

		name := fmt.Sprintf("%s.%s", structName, metodo.Name)
		description := et.Json{
			"inputs":  inputs,
			"outputs": outputs,
		}
		result[name] = description
		rpcs[name] = description

		logs.Logf("rpc", "RPC:/%s/%s", host, name)
	}

	err := rpc.Register(services)
	if err != nil {
		return nil, err
	}

	return result, nil
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
* CallRpc: Calls a remote procedure
* @param address string, method string, args any, reply any
* @return error
**/
func Call(address string, method string, args any, reply any) error {
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
* Close
**/
func Close() {
	logs.Log("Rpc", `Shutting down server...`)
}
