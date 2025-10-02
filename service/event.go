package service

import (
	"github.com/cgalvisleon/et/aws"
	"github.com/cgalvisleon/et/brevo"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

/**
* EventSendSms
* @param projectId string, contactNumbers []string, content string, params []et.Json, tp TpMessage, createdBy string
* @response string
**/
func EventSendSms(projectId string, contactNumbers []string, content string, params []et.Json, tp TpMessage, createdBy string) string {
	serviceId := New(projectId, "sms", "Send SMS message", createdBy)
	event.Work(EVENT_SEND_SMS, et.Json{
		"projectId":      projectId,
		"serviceId":      serviceId,
		"contactNumbers": contactNumbers,
		"content":        content,
		"params":         params,
		"type":           tp.String(),
		"createdBy":      createdBy,
	})

	return serviceId
}

/**
* EventSendWhatsapp
* @param projectId, templateId string, contactNumbers []string, params []et.Json, tp TpMessage, createdBy string
* @response string
**/
func EventSendWhatsapp(projectId, templateId string, contactNumbers []string, params []et.Json, tp TpMessage, createdBy string) string {
	serviceId := New(projectId, "whatsapp", "Send Whatsapp message", createdBy)
	event.Work(EVENT_SEND_WHATSAPP, et.Json{
		"projectId":      projectId,
		"serviceId":      serviceId,
		"templateId":     templateId,
		"contactNumbers": contactNumbers,
		"params":         params,
		"type":           tp.String(),
		"createdBy":      createdBy,
	})

	return serviceId
}

/**
* EventSendEmail
* @param projectId string, sender et.Json, to []et.Json, subject string, htmlContent string, params et.Json, tp TpMessage, createdBy string
* @response string
**/
func EventSendEmail(projectId string, sender et.Json, to []et.Json, subject string, htmlContent string, params et.Json, tp TpMessage, createdBy string) string {
	serviceId := New(projectId, "email", "Send email message", createdBy)
	event.Work(EVENT_SEND_EMAIL, et.Json{
		"projectId":   projectId,
		"serviceId":   serviceId,
		"sender":      sender,
		"to":          to,
		"subject":     subject,
		"htmlContent": htmlContent,
		"params":      params,
		"type":        tp.String(),
		"createdBy":   createdBy,
	})

	return serviceId
}

/**
* LoadEventSend
* @response error
**/
func LoadEventSend() {
	err := event.Subscribe(EVENT_SEND_SMS, eventSendSms)
	if err != nil {
		logs.Errorf(packageName, err.Error())
	}

	err = event.Subscribe(EVENT_SEND_WHATSAPP, eventSendWhatsapp)
	if err != nil {
		logs.Errorf(packageName, err.Error())
	}

	err = event.Subscribe(EVENT_SEND_EMAIL, eventSendEmail)
	if err != nil {
		logs.Errorf(packageName, err.Error())
	}
}

/**
* eventSendSms
* @param m event.Message
**/
func eventSendSms(m event.Message) {
	data := m.Data
	projectId := data.String("projectId")
	serviceId := data.String("serviceId")
	contactNumbers := data.ArrayStr("contactNumbers")
	content := data.String("content")
	params := data.ArrayJson("params")
	tp := data.String("type")
	createdBy := data.String("createdBy")
	result, err := aws.SendSMS(contactNumbers, content, params, tp)
	if err != nil {
		SetStatus(serviceId, STATUS_ERROR, et.Json{
			"error": err.Error(),
		})
		return
	}

	SetStatus(serviceId, STATUS_SUCCESS, et.Json{
		"context": et.Json{
			"projectId":      projectId,
			"serviceId":      serviceId,
			"contactNumbers": contactNumbers,
			"content":        content,
			"params":         params,
			"type":           tp,
			"createdBy":      createdBy,
			"sender":         "AWS SNS",
			"result":         result,
		},
	})

	logs.Logf(packageName, "eventSendSms: %s", data.ToString())
}

/**
* eventSendWhatsapp
* @param m event.Message
**/
func eventSendWhatsapp(m event.Message) {
	data := m.Data
	projectId := data.String("projectId")
	serviceId := data.String("serviceId")
	templateId := data.String("templateId")
	contactNumbers := data.ArrayStr("contactNumbers")
	params := data.ArrayJson("params")
	tp := data.String("type")
	createdBy := data.String("createdBy")
	result, err := brevo.SendWhatsapp(contactNumbers, templateId, params, tp)
	if err != nil {
		SetStatus(serviceId, STATUS_ERROR, et.Json{
			"error": err.Error(),
		})
		return
	}

	SetStatus(serviceId, STATUS_SUCCESS, et.Json{
		"context": et.Json{
			"projectId":      projectId,
			"serviceId":      serviceId,
			"templateId":     templateId,
			"contactNumbers": contactNumbers,
			"params":         params,
			"type":           tp,
			"createdBy":      createdBy,
			"sender":         "Brevo",
			"result":         result,
		},
	})

	logs.Logf(packageName, "eventSendWhatsapp: %s", data.ToString())
}

/**
* eventSendEmail
* @param m event.Message
**/
func eventSendEmail(m event.Message) {
	data := m.Data
	projectId := data.String("projectId")
	serviceId := data.String("serviceId")
	from := data.Json("from")
	to := data.ArrayJson("to")
	subject := data.String("subject")
	htmlContent := data.String("htmlContent")
	params := data.Json("params")
	tp := data.String("type")
	createdBy := data.String("createdBy")
	result, err := brevo.SendEmail(from, to, subject, htmlContent, params, tp)
	if err != nil {
		SetStatus(serviceId, STATUS_ERROR, et.Json{
			"error": err.Error(),
		})
		return
	}

	SetStatus(serviceId, STATUS_SUCCESS, et.Json{
		"context": et.Json{
			"projectId":   projectId,
			"serviceId":   serviceId,
			"from":        from,
			"to":          to,
			"subject":     subject,
			"htmlContent": htmlContent,
			"params":      params,
			"type":        tp,
			"createdBy":   createdBy,
			"sender":      "Brevo",
			"result":      result,
		},
	})

	logs.Logf(packageName, "eventSendEmail: %s", data.ToString())
}
