package jrpc

import (
	"errors"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
)

var (
	ErrorRpcNotConnected = errors.New("rpc not connected")
	pkg                  *Package
)

/**
* GetSolver
* @param method string
* @return (*Solver, error)
**/
func GetSolver(method string) (*Solver, error) {
	solver, ok := pkg.Solvers[method]
	if !ok {
		return nil, errors.New("solver not found")
	}
	return solver, nil
}

/**
* Close
**/
func Close() {
	logs.Log("Rpc", `Shutting down server...`)
}

/**
* listRouters
* @return []et.Json
* @return error
**/
func listRouters() ([]et.Json, error) {
	result := []et.Json{}
	for name, pkg := range rpcs {
		result = append(result, et.Json{
			"name": name,
			"pkg":  pkg,
		})
	}

	return result, nil
}

/**
* HttpListRouters
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpListRouters(w http.ResponseWriter, r *http.Request) {
	item, err := listRouters()
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
	}

	result := et.Items{}
	result.Add(item...)
	response.ITEMS(w, r, http.StatusOK, result)
}
