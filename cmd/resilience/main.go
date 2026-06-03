package main

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/resilience"
	"github.com/cgalvisleon/et/utility"
)

/**
* main
* @return void
**/
func main() {
	resilience, err := resilience.New(nil)
	if err != nil {
		logs.Panic(err)
	}
	ins := resilience.LoadInstance("", "suma", "func suma", "", 3, 3*time.Second, et.Json{}, "", "", suma, 1, 2)
	ins.Run()

	utility.AppWait()
}

func suma(a, b int) (int, error) {
	return a + b, errors.New("Error in suma")
}
