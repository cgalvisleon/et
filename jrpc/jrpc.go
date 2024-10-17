package jrpc

import (
	"encoding/json"
	"net"
	"net/rpc"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/strs"
)

type Router struct {
	key         string
	PackageName string             `json:"packageName"`
	Host        string             `json:"host"`
	Port        int                `json:"port"`
	Solvers     map[string]et.Json `json:"routes"`
}

var conn *Router

/**
* NewServer
**/
func StartServer() {
	_, err := cache.Load()
	if err != nil {
		return
	}

	address := strs.Format(`:%d`, conn.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logs.Fatal(err)
	}

	logs.Logf("Rpc", `Running on %s`, listener.Addr())
	for {
		conn, err := listener.Accept()
		if err != nil {
			logs.Panicf("Error al aceptar la conexión:%s", err.Error())
			continue
		}
		go rpc.ServeConn(conn)
	}
}

/**
* load
**/
func Load() {
	if conn != nil {
		return
	}

	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}

	port := envar.GetInt(4200, "RPC_PORT")

	conn = &Router{
		key:     "rpc-routes",
		Host:    host,
		Port:    port,
		Solvers: map[string]et.Json{},
	}
}

/**
* getRouters
* @param name string
* @return []*Router
* @return error
**/
func getRouters(name string) ([]*Router, error) {
	routers := make([]*Router, 0)
	jsonRoutes, err := json.Marshal(routers)
	if err != nil {
		return nil, err
	}

	strRoutes, err := cache.Get(name, string(jsonRoutes))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(strRoutes), &routers)
	if err != nil {
		return nil, err
	}

	return routers, nil
}

/**
* setRoutes
* @param name string
* @param routers []*Router
* @return error
**/
func setRoutes(name string, routers []*Router) error {
	bt, err := json.Marshal(routers)
	if err != nil {
		return err
	}

	cache.Set(name, string(bt), 0)

	return nil
}

/**
* UnMount
* @return error
**/
func UnMount() error {
	routers, err := getRouters(conn.key)
	if err != nil {
		return logs.Alert(err)
	}

	idx := slices.IndexFunc(routers, func(e *Router) bool { return e.Host == conn.Host && e.Port == conn.Port })
	if idx != -1 {
		routers = append(routers[:idx], routers[idx+1:]...)
	}

	err = setRoutes(conn.key, routers)
	if err != nil {
		return logs.Alert(err)
	}

	return nil
}

/**
* Save
* @return error
**/
func (r *Router) Save() error {
	routers, err := getRouters(r.key)
	if err != nil {
		return err
	}

	idx := slices.IndexFunc(routers, func(e *Router) bool { return e.Host == r.Host && e.Port == r.Port })
	if idx == -1 {
		routers = append(routers, r)
	}

	err = setRoutes(r.key, routers)
	if err != nil {
		return err
	}

	return nil
}

/**
* Mount
* @param host string
* @param port int
* @param service any
* @param packageName string
**/
func Mount(services any, packageName string) error {
	Load()
	tipoStruct := reflect.TypeOf(services)
	structName := tipoStruct.String()
	list := strings.Split(structName, ".")
	structName = list[len(list)-1]
	conn.PackageName = packageName
	for i := 0; i < tipoStruct.NumMethod(); i++ {
		metodo := tipoStruct.Method(i)
		numInputs := metodo.Type.NumIn()
		numOutputs := metodo.Type.NumOut()

		inputs := []string{}
		for i := 0; i < numInputs; i++ {
			inputs = append(inputs, metodo.Type.In(i).String())
		}

		outputs := []string{}
		for o := 0; o < numOutputs; o++ {
			outputs = append(outputs, metodo.Type.Out(o).String())
		}

		name := metodo.Name
		path := strs.Format(`%s.%s`, structName, name)
		conn.Solvers[path] = et.Json{
			"inputs":  inputs,
			"outputs": outputs,
		}
	}

	rpc.Register(services)

	return conn.Save()
}

/**
* GetSolver
* @param method string
* @return *Solver
* @return error
**/
func GetSolver(method string) (*Router, error) {
	routers, err := getRouters(conn.key)
	if err != nil {
		return nil, err
	}

	var result *Router
	for _, router := range routers {
		if router.Solvers[method] != nil {
			result = router
			break
		}
	}

	return result, nil
}

/**
* GetRouters
* @return et.Items
* @return error
**/
func GetRouters() (et.Items, error) {
	var result = et.Items{Result: []et.Json{}}
	routes, err := getRouters(conn.key)
	if err != nil {
		return et.Items{}, err
	}

	for _, route := range routes {
		n := 0
		_routes := []et.Json{}
		for k, v := range route.Solvers {
			n++
			_routes = append(_routes, et.Json{
				"method":  k,
				"inputs":  v["inputs"],
				"outputs": v["outputs"],
			})
		}

		result.Result = append(result.Result, et.Json{
			"packageName": route.PackageName,
			"host":        route.Host,
			"port":        route.Port,
			"count":       n,
			"routes":      _routes,
		})
		result.Ok = true
		result.Count++
	}

	return result, nil
}

/**
* Call
* @param method string
* @param data et.Json
* @return et.Item
* @return error
**/
func Call(method string, data et.Json) (et.Item, error) {
	metric := middleware.NewRpcMetric(method)
	var result = et.Item{Result: et.Json{}}
	solver, err := GetSolver(method)
	if err != nil {
		return result, err
	}

	if solver == nil {
		return result, logs.NewError(ERR_METHOD_NOT_FOUND)
	}

	address := strs.Format(`%s:%d`, solver.Host, solver.Port)
	metric.CallSearchTime()
	metric.SetAddress(address)

	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return et.Item{}, err
	}
	defer client.Close()

	err = client.Call(method, data, &result)
	if err != nil {
		return et.Item{}, err
	}

	metric.DoneRpc(result)

	return result, nil
}

func Close() {
	if conn != nil {
		UnMount()
	}

	logs.Log("Rpc", `Shutting down server...`)
}
