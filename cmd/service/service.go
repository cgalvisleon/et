package main

import "github.com/cgalvisleon/et/logs"

type Server struct {
	Name string
}

func New() (*Server, error) {
	// Create a new server

	logs.Logf("Server", "Create a new service")
	result := &Server{
		Name: "Server",
	}

	return result, nil
}

func (serv *Server) Close() error {
	return nil
}

func (serv *Server) Start() {
	// Start service
	go func() {

	}()

	<-make(chan struct{})
}
