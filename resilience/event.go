package resilience

import (
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

const (
	EVENT_RESILIENCE_STATUS  = "resilience:status"
	EVENT_RESILIENCE_STOP    = "resilience:stop"
	EVENT_RESILIENCE_RESTART = "resilience:restart"
	EVENT_RESILIENCE_FAILED  = "resilience:failed"
)

/**
* initEvents
**/
func initEvents() {
	err := event.Subscribe(EVENT_RESILIENCE_STOP, eventStop)
	if err != nil {
		logs.Error(packageName, err)
	}

	err = event.Subscribe(EVENT_RESILIENCE_RESTART, eventRestart)
	if err != nil {
		logs.Error(packageName, err)
	}

}

/**
* eventStop
* @param m event.Message
**/
func eventStop(m event.Message) {
	data := m.Data
	id := data.Str("id")
	if id == "" {
		logs.Errorf(packageName, MSG_ID_REQUIRED)
		return
	}

	Stop(id)
	logs.Log(packageName, "eventStop:", data.ToString())
}

/**
* eventRestart
* @param m event.Message
**/
func eventRestart(m event.Message) {
	data := m.Data
	id := data.Str("id")
	if id == "" {
		logs.Errorf(packageName, MSG_ID_REQUIRED)
		return
	}

	Restart(id)
	logs.Log(packageName, "eventRestart:", data.ToString())
}
