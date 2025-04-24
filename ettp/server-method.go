package ettp

import (
	"net/http"
)

/**
* Connect
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *Server) Connect(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setHandlerFunc(CONNECT, path, handlerFn, packageName)
}

/**
* Delete
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *Server) Delete(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setHandlerFunc(DELETE, path, handlerFn, packageName)
}

/**
* Get
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *Server) Get(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setHandlerFunc(GET, path, handlerFn, packageName)
}

/**
* Head
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *Server) Head(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setHandlerFunc(HEAD, path, handlerFn, packageName)
}

/**
* Options
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *Server) Options(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setHandlerFunc(OPTIONS, path, handlerFn, packageName)
}

/**
* Patch
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *Server) Patch(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setHandlerFunc(PATCH, path, handlerFn, packageName)
}

/**
* Post
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *Server) Post(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setHandlerFunc(POST, path, handlerFn, packageName)
}

/**
* Put
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *Server) Put(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setHandlerFunc(PUT, path, handlerFn, packageName)
}

/**
* Trace
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *Server) Trace(path string, handlerFn http.HandlerFunc, packageName string) {
	s.setHandlerFunc(TRACE, path, handlerFn, packageName)
}
