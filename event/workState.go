package event

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type WorkStatus string

const (
	StatusPending    WorkStatus = "pending"
	StatusProcessing WorkStatus = "processing"
	StatusCompleted  WorkStatus = "completed"
	StatusFailed     WorkStatus = "failed"
)

/**
* Work
* @param event string, data et.Json
* @return et.Json
**/
func Work(event string, data et.Json) et.Json {
	id := reg.GenULID("work")
	work := et.Json{
		"created_at": timezone.NowTime(),
		"status":     StatusPending,
		"id":         id,
		"event":      event,
		"data":       data,
	}

	go Publish(EVENT_WORK, work)
	go Publish(event, work)

	return work
}

/**
* State
* @param id string, status WorkStatus
**/
func State(id string, status WorkStatus, data et.Json) {
	now := timezone.Now()
	work := et.Json{
		"update_at": now,
		"id":        id,
		"status":    status,
		"data":      data,
	}

	go Publish(EVENT_WORK_STATE, work)
	go Publish(id, work)
}
