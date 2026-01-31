package jrpc

import (
	"encoding/gob"
	"fmt"
	"net"
	"net/rpc"
	"reflect"
	"runtime"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

var (
	pkg *Package
	os  = ""
)

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
		result[name] = et.Json{
			"inputs":  inputs,
			"outputs": outputs,
		}

		logs.Logf("rpc", "RPC:/%s/%s", host, name)
	}

	if err := rpc.Register(services); err != nil {
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

	return nil
}

func LoadTo(name, host string, port int) (*Package, error) {
	if !utility.ValidStr(name, 1, []string{"", ""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	if !utility.ValidStr(host, 1, []string{"", ""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "host")
	}

	if !utility.ValidInt(port, []int{1, 65535}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "port")
	}

	name = strs.DaskSpace(name)
	result := NewPackage(name, host, port)

	return result, nil
}

/**
* load
**/
func Load(name string) error {
	if !slices.Contains([]string{"linux", "darwin", "windows"}, os) {
		return nil
	}

	if pkg != nil {
		return nil
	}

	err := cache.Load()
	if err != nil {
		return err
	}

	err = envar.Validate([]string{
		"RPC_PORT",
	})
	if err != nil {
		return err
	}

	host := envar.GetStr("RPC_HOST", "localhost")
	port := envar.GetInt("RPC_PORT", 4200)
	pkg, err = LoadTo(name, host, port)
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

func init() {
	os = runtime.GOOS
	gob.Register(map[string]interface{}{})
	gob.Register(et.Json{})
	gob.Register(et.Item{})
	gob.Register(et.Items{})
	gob.Register(et.List{})
}
