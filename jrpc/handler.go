package jrpc

import (
	"encoding/gob"
	"net/http"
	"net/rpc"
	"slices"

	"github.com/celsiainternet/elvis/et"
	"github.com/celsiainternet/elvis/logs"
	"github.com/celsiainternet/elvis/middleware"
	"github.com/celsiainternet/elvis/response"
	"github.com/celsiainternet/elvis/strs"
)

/**
* GetRouters
* @return et.Items
* @return error
**/
func GetRouters() (et.Items, error) {
	var result = et.Items{Result: []et.Json{}}
	routes, err := getRouters()
	if err != nil {
		return et.Items{}, err
	}

	for _, route := range routes {
		_routes := []et.Json{}
		for k, v := range route.Solvers {
			_routes = append(_routes, et.Json{
				"method":  k,
				"inputs":  v.Inputs,
				"outputs": v.Output,
			})
		}

		result.Result = append(result.Result, et.Json{
			"packageName": route.Name,
			"host":        route.Host,
			"port":        route.Port,
			"count":       len(_routes),
			"routes":      _routes,
		})
		result.Ok = true
		result.Count++
	}

	return result, nil
}

/**
* CallJson
* @param method string
* @param data et.Json
* @return et.Json
* @return error
**/
func CallJson(method string, data et.Json) (et.Json, error) {
	var result et.Json
	metric := middleware.NewRpcMetric(method)
	solver, err := GetSolver(method)
	if err != nil {
		return result, err
	}

	address := strs.Format(`%s:%d`, solver.Host, solver.Port)
	metric.CallSearchTime()
	metric.ClientIP = address

	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return result, err
	}
	defer client.Close()

	err = client.Call(solver.Method, data, &result)
	if err != nil {
		return result, logs.Alert(err)
	}

	metric.DoneRpc(result.ToString())

	return result, nil
}

/**
* CallItem
* @param method string
* @param data et.Json
* @return et.Item
* @return error
**/
func CallItem(method string, data et.Json) (et.Item, error) {
	var result et.Item
	metric := middleware.NewRpcMetric(method)
	solver, err := GetSolver(method)
	if err != nil {
		return result, err
	}

	address := strs.Format(`%s:%d`, solver.Host, solver.Port)
	metric.CallSearchTime()
	metric.ClientIP = address

	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return result, err
	}
	defer client.Close()

	err = client.Call(solver.Method, data, &result)
	if err != nil {
		return result, logs.Alert(err)
	}

	metric.DoneRpc(result.ToJson().ToString())

	return result, nil
}

/**
* CallItems
* @param method string
* @param data et.Json
* @return et.Item
* @return error
**/
func CallItems(method string, data et.Json) (et.Items, error) {
	var result et.Items
	metric := middleware.NewRpcMetric(method)
	solver, err := GetSolver(method)
	if err != nil {
		return result, err
	}

	address := strs.Format(`%s:%d`, solver.Host, solver.Port)
	metric.CallSearchTime()
	metric.ClientIP = address

	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return result, err
	}
	defer client.Close()

	err = client.Call(solver.Method, data, &result)
	if err != nil {
		return result, logs.Alert(err)
	}

	metric.DoneRpc(result.ToJson().ToString())

	return result, nil
}

/**
* CallList
* @param method string
* @param data et.Json
* @return et.List
* @return error
**/
func CallList(method string, data et.Json) (et.List, error) {
	var result et.List
	metric := middleware.NewRpcMetric(method)
	solver, err := GetSolver(method)
	if err != nil {
		return result, err
	}

	address := strs.Format(`%s:%d`, solver.Host, solver.Port)
	metric.CallSearchTime()
	metric.ClientIP = address

	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return result, err
	}
	defer client.Close()

	err = client.Call(solver.Method, data, &result)
	if err != nil {
		return result, logs.Alert(err)
	}

	metric.DoneRpc(result.ToJson().ToString())

	return result, nil
}

/**
* CallPermitios
* @param method string
* @param data et.Json
* @return map[string]bool
* @return error
**/
func CallPermitios(method string, data et.Json) (map[string]bool, error) {
	metric := middleware.NewRpcMetric(method)
	solver, err := GetSolver(method)
	if err != nil {
		return map[string]bool{}, err
	}

	address := strs.Format(`%s:%d`, solver.Host, solver.Port)
	metric.CallSearchTime()
	metric.ClientIP = address

	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return map[string]bool{}, err
	}
	defer client.Close()

	var result map[string]bool
	err = client.Call(solver.Method, data, &result)
	if err != nil {
		return map[string]bool{}, logs.Error(err)
	}

	metric.DoneRpc(result)

	return result, nil
}

/**
* DeleteRouters
* @param host string
* @param packageName string
* @return et.Item
* @return error
**/
func DeleteRouters(host, packageName string) (et.Item, error) {
	routers, err := getRouters()
	if err != nil {
		return et.Item{}, err
	}

	idx := slices.IndexFunc(routers, func(e *Package) bool { return e.Host == host && e.Name == packageName })
	if idx == -1 {
		return et.Item{}, logs.Errorm(MSG_PACKAGE_NOT_FOUND)
	} else {
		routers = append(routers[:idx], routers[idx+1:]...)
	}

	err = setRoutes(routers)
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"message": MSG_PACKAGE_DELETE,
		},
	}, nil
}

/**
* HttpCallRPC
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpCallRPC(w http.ResponseWriter, r *http.Request) {
	body, _ := response.GetBody(r)
	method := body.ValStr("", "method")
	data := body.Json("data")
	result, err := CallItem(method, data)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
	}

	response.JSON(w, r, http.StatusOK, result)
}

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register(map[string]bool{})
	gob.Register(map[string]string{})
	gob.Register(map[string]int{})
	gob.Register(et.Json{})
	gob.Register([]et.Json{})
	gob.Register(et.Item{})
	gob.Register(et.Items{})
	gob.Register(et.List{})
}
