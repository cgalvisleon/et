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
	res, err := resilience.New(nil)
	if err != nil {
		logs.Panic(err)
	}

	ins := res.LoadInstance(resilience.Params{
		TenantId:      "1234567890",
		Id:            "suma",
		Tag:           "func suma",
		Description:   "",
		OwnerId:       "",
		TotalAttempts: 3,
		Interval:      3 * time.Second,
		Tags:          et.Json{},
		UserId:        "1234567890",
		Fn:            suma,
		FnArgs:        []interface{}{1, 2},
	})
	ins.Run("1234567890")

	utility.AppWait()
}

func suma(a, b int) (int, error) {
	return a + b, errors.New("Error in suma")
}
