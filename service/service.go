package service

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
)

// Events
const (
	packageName                  = "Service"
	EVENT_SEND_SMS               = "send:sms"
	EVENT_SEND_WHATSAPP          = "send:whatsapp"
	EVENT_SEND_EMAIL             = "send:email"
	EVENT_SEND_EMAIL_BY_TEMPLATE = "send:email:template"
	EVENT_SEND_PUSH              = "send:push"
	SERVICE_STATUS               = "service:status"
)

// Status
const (
	STATUS_PENDING   = "pending"
	STATUS_SUCCESS   = "success"
	STATUS_FAILED    = "failed"
	STATUS_WORKING   = "working"
	STATUS_ERROR     = "error"
	STATUS_CANCELLED = "cancelled"
	STATUS_EXPIRED   = "expired"
)

/**
* New
* @param tenantId, kind, description, clientId string
* @response string
**/
func New(tenantId, kind, description, clientId string) string {
	now := utility.Now()
	result := reg.GenULIDI("service")
	data := et.Json{
		"created_at":  now,
		"service_id":  result,
		"tenant_id":   tenantId,
		"client_id":   clientId,
		"kind":        kind,
		"description": description,
		"status":      STATUS_PENDING,
		"context":     et.Json{},
	}

	event.Work(SERVICE_STATUS, data)
	cache.SetH(result, data, 1)
	return result
}

/**
* GetById
* @param serviceId string
* @response et.Json, error
**/
func GetById(serviceId string) (et.Json, error) {
	return cache.GetJson(serviceId)
}

/**
* SetStatus
* @param serviceId, status string, context et.Json
**/
func SetStatus(serviceId, status string, context et.Json) error {
	if status == "" {
		status = STATUS_PENDING
	}

	data, err := GetById(serviceId)
	if err != nil {
		return err
	}

	data["updated_at"] = utility.Now()
	data["status"] = status
	data["context"] = context

	event.Work(SERVICE_STATUS, data)
	cache.SetH(serviceId, data, 1)
	return nil
}
