package event

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type WorkStatus int

const (
	WorkStatusPending WorkStatus = iota
	WorkStatusAccepted
	WorkStatusProcessing
	WorkStatusCompleted
	WorkStatusFailed
)

/**
* String
* @return string
**/
func (s WorkStatus) String() string {
	switch s {
	case WorkStatusPending:
		return "Pending"
	case WorkStatusAccepted:
		return "Accepted"
	case WorkStatusProcessing:
		return "Processing"
	case WorkStatusCompleted:
		return "Completed"
	case WorkStatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

/**
* ToWorkStatus
* @param int n
* @return WorkStatus
**/
func ToWorkStatus(n int) WorkStatus {
	switch n {
	case 0:
		return WorkStatusPending
	case 1:
		return WorkStatusAccepted
	case 2:
		return WorkStatusProcessing
	case 3:
		return WorkStatusCompleted
	case 4:
		return WorkStatusFailed
	default:
		return WorkStatusPending
	}
}

/**
* Work
* @param event string
* @param data et.Json
**/
func Work(event string, data et.Json) et.Json {
	work := et.Json{
		"created_at": timezone.Now(),
		"_id":        reg.GenId("work"),
		"from_id":    conn.Id,
		"event":      event,
		"data":       data,
	}

	go Publish(EVENT_WORK, work)
	go Publish(event, work)

	return work
}

/**
* WorkState
* @param work_id string
* @param status WorkStatus
* @param data et.Json
**/
func WorkState(work_id string, status WorkStatus, data et.Json) {
	work := et.Json{
		"update_at": timezone.Now(),
		"_id":       work_id,
		"from_id":   conn.Id,
		"status":    status.String(),
		"data":      data,
	}
	switch status {
	case WorkStatusPending:
		work["pending_at"] = utility.Now()
	case WorkStatusAccepted:
		work["accepted_at"] = utility.Now()
	case WorkStatusProcessing:
		work["processing_at"] = utility.Now()
	case WorkStatusCompleted:
		work["completed_at"] = utility.Now()
	case WorkStatusFailed:
		work["failed_at"] = utility.Now()
	}

	go Publish(EVENT_WORK_STATE, work)
}
