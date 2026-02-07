package jrpc

type Solver struct {
	Method string   `json:"method"`
	Inputs []string `json:"inputs"`
	Output []string `json:"output"`
}
