package main

import (
	"github.com/cgalvisleon/et/resilience"
)

/**
* main
* @return void
**/
func main() {
	resilience, err := resilience.New(nil)
	if err != nil {
		panic(err)
	}
}
