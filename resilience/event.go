package resilience

import (
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/event"
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
		console.Error(err)
	}

	err = event.Subscribe(EVENT_RESILIENCE_RESTART, eventRestart)
	if err != nil {
		console.Error(err)
	}

}

/**
* eventStop
* @param m event.EvenMessage
**/
func eventStop(m event.Message) {
	data := m.Data
	id := data.Str("id")
	if id == "" {
		console.Errorm(MSG_ID_REQUIRED)
		return
	}

	Stop(id)
	console.Log("eventStop:", data.ToString())
}

/**
* eventRestart
* @param m event.EvenMessage
**/
func eventRestart(m event.Message) {
	data := m.Data
	id := data.Str("id")
	if id == "" {
		console.Errorm(MSG_ID_REQUIRED)
		return
	}

	Restart(id)
	console.Log("eventRestart:", data.ToString())
}
