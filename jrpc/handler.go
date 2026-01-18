package jrpc

import (
	"fmt"
	"net/http"
	"net/rpc"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/response"
)

/**
* Mount
* @param services any
* @return error
**/
func Mount(services any) error {
	if pkg == nil {
		return logs.Alertm(msg.MSG_PACKAGE_NOT_FOUND)
	}

	return pkg.mount(services)
}

/**
* listRouters
* @return et.Items
* @return error
**/
func listRouters() (et.Items, error) {
	var result = et.Items{Result: []et.Json{}}
	packages, err := getPackages()
	if err != nil {
		return et.Items{}, err
	}

	for _, pkg := range packages {
		result.Add(pkg.Describe())
	}

	return result, nil
}

/**
* call
* @param method string, args et.Json, reply interface{}
* @return error
**/
func call(method string, args any, reply any) error {
	err := cache.Load()
	if err != nil {
		return err
	}

	metric := middleware.NewRpcMetric(method)
	solver, err := getSolver(method)
	if err != nil {
		return err
	}

	address := fmt.Sprintf(`%s:%d`, solver.Host, solver.Port)
	metric.CallSearchTime()
	metric.RemoteAddr = address

	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return err
	}
	defer client.Close()

	methodName := fmt.Sprintf(`%s.%s`, solver.StructName, solver.Method)
	err = client.Call(methodName, args, reply)
	if err != nil {
		return err
	}

	metric.DoneRpc(reply)

	return nil
}

/**
* CallJson
* @param method string, args et.Json
* @return et.Json, error
**/
func CallJson(method string, args et.Json) (et.Json, error) {
	var result et.Json
	err := call(method, args, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

/**
* CallItem
* @param method string, args et.Json
* @return et.Item, error
**/
func CallItem(method string, args et.Json) (et.Item, error) {
	var result et.Item
	err := call(method, args, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

/**
* CallItems
* @param method string, args et.Json
* @return et.Items, error
**/
func CallItems(method string, args et.Json) (et.Items, error) {
	var result et.Items
	err := call(method, args, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

/**
* CallList
* @param method string, args et.Json
* @return et.List, error
**/
func CallList(method string, args et.Json) (et.List, error) {
	var result et.List
	err := call(method, args, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

/**
* HttpListRouters
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpListRouters(w http.ResponseWriter, r *http.Request) {
	result, err := listRouters()
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
	}

	response.ITEMS(w, r, http.StatusOK, result)
}

/**
* HttpCallItem
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpCalcItem(w http.ResponseWriter, r *http.Request) {
	body, _ := response.GetBody(r)
	method := body.ValStr("", "method")
	data := body.Json("data")
	result, err := CallItem(method, data)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
	}

	response.ITEM(w, r, http.StatusOK, result)
}
