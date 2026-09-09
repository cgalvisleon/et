package jrpc

import (
	"errors"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
)

var (
	ErrorRpcNotConnected   = errors.New("rpc not connected")
	ErrorPackageNotMounted = errors.New("rpc package not mounted")
	pkg                    *Package
)

/**
* GetSolver
* @param method string
* @return (*Solver, error)
**/
func GetSolver(method string) (*Solver, error) {
	mu.RLock()
	defer mu.RUnlock()

	if pkg == nil {
		return nil, ErrorPackageNotMounted
	}

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
	mu.Lock()
	l := listener
	mu.Unlock()

	if l != nil {
		l.Close()
	}
	logs.Log("Rpc", `Shutting down server...`)
}

/**
* listRouters
* @return []et.Json
* @return error
**/
func listRouters() ([]et.Json, error) {
	mu.RLock()
	defer mu.RUnlock()

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
