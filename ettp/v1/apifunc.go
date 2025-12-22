package ettp

import (
	"fmt"
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
* @param method, path string, h http.HandlerFunc, packageName string
* @return *ApiFunc
**/
func NewApiFunc(method, path string, h http.HandlerFunc, packageName string) *ApiFunc {
	key := fmt.Sprintf("%s:%s", method, path)
	return &ApiFunc{
		Id:          key,
		Method:      method,
		Path:        path,
		HandlerFn:   h,
		PackageName: packageName,
	}
}
