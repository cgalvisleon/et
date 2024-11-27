package jrpc

import (
	"net/rpc"
	"reflect"
	"slices"
	"strings"

	"github.com/celsiainternet/elvis/logs"
	"github.com/celsiainternet/elvis/strs"
)

type Solver struct {
	PackageName string   `json:"packageName"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Method      string   `json:"method"`
	Inputs      []string `json:"inputs"`
	Output      []string `json:"outputs"`
}

/**
* Mount
* @param host string
* @param port int
* @param service any
**/
func Mount(services any) error {
	if pkg == nil {
		return logs.Alertm(ERR_PACKAGE_NOT_FOUND)
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
		for i := 0; i < numInputs; i++ {
			inputs = append(inputs, metodo.Type.In(i).String())
		}

		outputs := []string{}
		for o := 0; o < numOutputs; o++ {
			outputs = append(outputs, metodo.Type.Out(o).String())
		}

		structName = strs.DaskSpace(structName)
		name := strs.DaskSpace(metodo.Name)
		method := strs.Format(`%s.%s`, structName, name)
		key := strs.Format(`%s.%s.%s`, pkg.Name, structName, name)
		solver := &Solver{
			PackageName: pkg.Name,
			Host:        pkg.Host,
			Port:        pkg.Port,
			Method:      method,
			Inputs:      inputs,
			Output:      outputs,
		}
		pkg.Solvers[key] = solver
	}

	rpc.Register(services)

	return pkg.Save()
}

/**
* UnMount
* @return error
**/
func UnMount() error {
	if pkg == nil {
		return logs.Alertm(ERR_PACKAGE_NOT_FOUND)
	}

	routers, err := getRouters()
	if err != nil {
		return logs.Alert(err)
	}

	idx := slices.IndexFunc(routers, func(e *Package) bool { return e.Name == pkg.Name })
	if idx != -1 {
		routers = append(routers[:idx], routers[idx+1:]...)
	}

	err = setRoutes(routers)
	if err != nil {
		return logs.Alert(err)
	}

	return nil
}

/**
* GetSolver
* @param method string
* @return *Solver
* @return error
**/
func GetSolver(method string) (*Solver, error) {
	method = strings.TrimSpace(method)
	routers, err := getRouters()
	if err != nil {
		return nil, err
	}

	lst := strings.Split(method, ".")
	if len(lst) != 3 {
		return nil, logs.NewErrorf(ERR_METHOD_NOT_FOUND, method)
	}

	packageName := lst[0]
	idx := slices.IndexFunc(routers, func(e *Package) bool { return e.Name == packageName })
	if idx == -1 {
		return nil, logs.NewError(ERR_PACKAGE_NOT_FOUND)
	}

	router := routers[idx]
	solver := router.Solvers[method]

	if solver == nil {
		return nil, logs.NewErrorf(ERR_METHOD_NOT_FOUND, method)
	}

	return solver, nil
}
