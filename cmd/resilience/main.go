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
	res, err := resilience.New()
	if err != nil {
		logs.Panic(err)
	}

	ins, err := res.LoadInstance(resilience.Params{
		Id:            "suma",
		Tag:           "func suma",
		Description:   "",
		TotalAttempts: 3,
		Interval:      3 * time.Second,
		Tags:          et.Json{},
		Fn:            suma,
		FnArgs:        []interface{}{1, 2},
	})
	ins.Run()

	utility.AppWait()
}

func suma(a, b int) (int, error) {
	return a + b, errors.New("Error in suma")
}
