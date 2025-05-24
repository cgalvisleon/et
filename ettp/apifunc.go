package ettp

import (
	"net/http"
)

type ApiFunc struct {
	Id          string
	Method      string
	Path        string
	HandlerFn   http.HandlerFunc
	PackageName string
}

/**
* NewApiFunc
* @param id, method, path string, h http.HandlerFunc, packageName string
* @return *ApiFunc
**/
func NewApiFunc(id, method, path string, h http.HandlerFunc, packageName string) *ApiFunc {
	return &ApiFunc{
		Id:          id,
		Method:      method,
		Path:        path,
		HandlerFn:   h,
		PackageName: packageName,
	}
}
