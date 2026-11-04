package jrpc

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/response"
)

var (
	ErrorRpcNotConnected   = errors.New("rpc not connected")
	ErrorPackageNotMounted = errors.New("rpc package not mounted")
	pkg                    *Package
	resolver               func(method string) (*Solver, error)
)

/**
* SetResolver: Where to find a method that this process did not mount (another service): GetSolver asks it after the
* local package. The service that keeps the catalog of the methods sets it
* @param fn func(method string) (*Solver, error)
**/
func SetResolver(fn func(method string) (*Solver, error)) {
	mu.Lock()
	resolver = fn
	mu.Unlock()
}

/**
* GetSolver: Where a method is served: the package mounted in this process or, without it there, the resolver
* @param method string
* @return (*Solver, error)
**/
func GetSolver(method string) (*Solver, error) {
	mu.RLock()
	var solver *Solver
	ok := false
	mounted := pkg != nil
	if mounted {
		solver, ok = pkg.Solvers[method]
	}
	fn := resolver
	mu.RUnlock()

	if ok {
		return solver, nil
	}

	if fn != nil {
		return fn(method)
	}

	if !mounted {
		return nil, ErrorPackageNotMounted
	}

	return nil, fmt.Errorf(msg.MSG_SOLVER_NOT_FOUND, method)
}

/**
* Methods: The methods mounted in this process, as GET /rpcs gives them: {packageName, method, host, port, inputs, output}
* @return []et.Json
**/
func Methods() []et.Json {
	mu.RLock()
	defer mu.RUnlock()

	result := []et.Json{}
	if pkg == nil {
		return result
	}

	for _, method := range slices.Sorted(maps.Keys(pkg.Solvers)) {
		solver := pkg.Solvers[method]
		result = append(result, et.Json{
			"packageName": pkg.Name,
			"method":      method,
			"host":        solver.Host,
			"port":        solver.Port,
			"inputs":      solver.Inputs,
			"output":      solver.Output,
		})
	}
	return result
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
* listRouters: Lists the mounted packages.
* @return []et.Json, error
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
		return
	}

	result := et.Items{}
	result.Add(item...)
	response.ITEMS(w, r, http.StatusOK, result)
}
