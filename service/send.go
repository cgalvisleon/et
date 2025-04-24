package service

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/utility"
)

type TpMessage int

const (
	TypeNotification TpMessage = iota
	TypeComercial
	TypeAutentication
)

func (tp TpMessage) String() string {
	return [...]string{"Notification", "Comercial", "Autentication"}[tp]
}

func IntToTpMessage(i int) TpMessage {
	return TpMessage(i)
}

/**
* GetId
* @param client_id, kind, description string
* @response string
**/
func GetId(client_id, kind, description string) string {
	now := utility.Now()
	result := utility.UUID()
	data := et.Json{
		"created_at":  now,
		"service_id":  result,
		"client_id":   client_id,
		"kind":        kind,
		"description": description,
	}
	event.Work("service/client", data)
	cache.SetH(result, data, 1)

	return result
}

/**
* SetStatus
* @param serviceId string, status et.Json
**/
func SetStatus(serviceId string, status et.Json) {
	event.Work("service/status", et.Json{
		"service_id": serviceId,
		"status":     status,
	})

	cache.SetH(serviceId, status, 1)
}

/**
* GetStatus
* @param serviceId string
* @response et.Json, error
**/
func GetStatus(serviceId string) (et.Json, error) {
	return cache.GetJson(serviceId)
}

/**
* SendSms
* @param projectId string, contactNumbers []string, content string, params []et.Json, tp TpMessage, createdBy string
* @response et.Json
**/
func SendSms(projectId string, contactNumbers []string, content string, params []et.Json, tp TpMessage, createdBy string) et.Json {
	serviceId := GetId(createdBy, "sms", "Send SMS message")
	result := event.Work("send/sms", et.Json{
		"projectId":      projectId,
		"serviceId":      serviceId,
		"contactNumbers": contactNumbers,
		"content":        content,
		"params":         params,
		"type":           tp,
		"createdBy":      createdBy,
	})

	result["serviceId"] = serviceId
	return result
}

/**
* SendWhatsapp
* @param projectId, string, templateId int, contactNumbers []string, params []et.Json, tp TpMessage, createdBy string
* @response et.Json
**/
func SendWhatsapp(projectId, string, templateId int, contactNumbers []string, params []et.Json, tp TpMessage, createdBy string) et.Json {
	serviceId := GetId(createdBy, "whatsapp", "Send Whatsapp message")
	result := event.Work("send/whatsapp", et.Json{
		"projectId":      projectId,
		"serviceId":      serviceId,
		"templateId":     templateId,
		"contactNumbers": contactNumbers,
		"params":         params,
		"type":           tp,
		"createdBy":      createdBy,
	})

	result["serviceId"] = serviceId
	return result
}

/**
* SendEmail
* @param projectId string, to []et.Json, subject string, htmlContent string, params []et.Json, tp TpMessage, createdBy string
* @response et.Json
**/
func SendEmail(projectId string, to []et.Json, subject string, htmlContent string, params []et.Json, tp TpMessage, createdBy string) et.Json {
	serviceId := GetId(createdBy, "email", "Send email message")
	result := event.Work("send/email", et.Json{
		"projectId":   projectId,
		"serviceId":   serviceId,
		"to":          to,
		"subject":     subject,
		"htmlContent": htmlContent,
		"params":      params,
		"type":        tp.String(),
		"createdBy":   createdBy,
	})

	result["serviceId"] = serviceId
	return result
}

/**
* SendEmailByTemplate
* @param projectId string, to []et.Json, subject string, templateId int, params []et.Json, tp TpMessage, createdBy string
* @response et.Json
**/
func SendEmailByTemplate(projectId string, to []et.Json, subject string, templateId int, params []et.Json, tp TpMessage, createdBy string) et.Json {
	serviceId := GetId(createdBy, "email", "Send Email By Template")
	result := event.Work("send/email/template", et.Json{
		"projectId":  projectId,
		"serviceId":  serviceId,
		"to":         to,
		"subject":    subject,
		"templateId": templateId,
		"params":     params,
		"type":       tp.String(),
		"createdBy":  createdBy,
	})

	result["serviceId"] = serviceId
	return result
}

/**
* SendPush
* @param projectId string, to []et.Json, subject string, content string, params []et.Json, tp TpMessage, createdBy string
* @response et.Json
**/
func SendPush(projectId string, to []et.Json, subject string, content string, params []et.Json, tp TpMessage, createdBy string) et.Json {
	serviceId := GetId(createdBy, "push", "Send Push message")
	result := event.Work("send/push", et.Json{
		"projectId": projectId,
		"serviceId": serviceId,
		"to":        to,
		"subject":   subject,
		"content":   content,
		"params":    params,
		"type":      tp,
		"createdBy": createdBy,
	})

	result["serviceId"] = serviceId
	return result
}
