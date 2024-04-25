package gateway

import (
	"github.com/cgalvisleon/elvis/console"
	"github.com/cgalvisleon/elvis/et"
	"github.com/cgalvisleon/elvis/event"
)

func initEvents() {
	console.LogK("Events", "Running svents stack")

	err := event.Stack("gateway/upsert", eventAction)
	if err != nil {
		console.Error(err)
	}

}

func eventAction(m event.CreatedEvenMessage) {
	data, err := et.ToJson(m.Data)
	if err != nil {
		console.Error(err)
	}

	method := data.Str("method")
	path := data.Str("path")
	resolve := data.Str("resolve")
	kind := data.ValStr("HTTP", "kind")
	stage := data.ValStr("default", "stage")
	packageName := data.Str("package")

	conn.http.AddRoute(method, path, resolve, kind, stage, packageName)

	console.LogKF("Api gateway", `[%s] %s - %s`, method, path, packageName)
}
