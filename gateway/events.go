package gateway

import (
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/message"
)

func initEvents() error {
	// Events
	err := event.Load()
	if err != nil {
		return err
	}

	/*
		err := event.Stack("gateway/upsert", eventAction)
		if err != nil {
			logs.Error(err)
		}
	*/

	logs.Log("Events", "Load events")

	return nil
}

func inbox(msg message.Message) {
	logs.Debug(msg.ToString())
}

/*
func eventAction(m event.CreatedEvenMessage) {
	data, err := et.ToJson(m.Data)
	if err != nil {
		logs.Error(err)
	}

	method := data.Str("method")
	path := data.Str("path")
	resolve := data.Str("resolve")
	kind := data.ValStr("HTTP", "kind")
	stage := data.ValStr("default", "stage")
	packageName := data.Str("package")

	conn.http.AddRoute(method, path, resolve, kind, stage, packageName)

	logs.LogKF("Api gateway", `[%s] %s - %s`, method, path, packageName)
}
*/
