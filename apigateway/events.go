package apigateway

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
)

func initEvents() {
	et.Log("Events", "Running svents stack")

	err := event.Stack("apigateway/upsert", eventAction)
	if err != nil {
		et.Error(err)
	}

}

func eventAction(m event.CreatedEvenMessage) {
	data, err := et.ToJson(m.Data)
	if err != nil {
		et.Error(err)
	}

	method := data.Str("method")
	path := data.Str("path")
	resolve := data.Str("resolve")
	kind := data.ValStr("HTTP", "kind")
	stage := data.ValStr("default", "stage")
	packageName := data.Str("package")

	AddRoute(method, path, resolve, kind, stage, packageName)

	et.Log("Event", m.Channel)
}
