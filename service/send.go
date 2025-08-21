package service

import (
	"github.com/cgalvisleon/et/aws"
	"github.com/cgalvisleon/et/brevo"
	"github.com/cgalvisleon/et/et"
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
* SendSms
* @param projectId string, contactNumbers []string, content string, params []et.Json, tp TpMessage, createdBy string
* @response et.Item, error
**/
func SendSms(projectId string, contactNumbers []string, content string, params []et.Json, tp TpMessage, createdBy string) (et.Items, error) {
	serviceId := New(projectId, "sms", "Send SMS message", createdBy)
	result, err := aws.SendSMS(contactNumbers, content, params, tp.String())
	if err != nil {
		SetStatus(serviceId, STATUS_ERROR, et.Json{
			"error": err.Error(),
		})
		return et.Items{}, err
	}

	SetStatus(serviceId, STATUS_SUCCESS, et.Json{
		"context": et.Json{
			"projectId":      projectId,
			"serviceId":      serviceId,
			"contactNumbers": contactNumbers,
			"content":        content,
			"params":         params,
			"type":           tp.String(),
			"createdBy":      createdBy,
			"sender":         "AWS SNS",
			"result":         result,
		},
	})

	return result, nil
}

/**
* SendWhatsapp
* @param projectId, templateId string, contactNumbers []string, params []et.Json, tp TpMessage, createdBy string
* @response et.Items, error
**/
func SendWhatsapp(projectId, templateId string, contactNumbers []string, params []et.Json, tp TpMessage, createdBy string) (et.Items, error) {
	serviceId := New(projectId, "whatsapp", "Send Whatsapp message", createdBy)
	result, err := brevo.SendWhatsapp(contactNumbers, templateId, params, tp.String())
	if err != nil {
		SetStatus(serviceId, STATUS_ERROR, et.Json{
			"error": err.Error(),
		})
		return et.Items{}, err
	}

	SetStatus(serviceId, STATUS_SUCCESS, et.Json{
		"context": et.Json{
			"projectId":      projectId,
			"serviceId":      serviceId,
			"templateId":     templateId,
			"contactNumbers": contactNumbers,
			"params":         params,
			"type":           tp.String(),
			"createdBy":      createdBy,
			"sender":         "Brevo",
			"result":         result,
		},
	})

	return result, nil
}

/**
* SendEmail
* @param projectId string, from et.Json, to []et.Json, subject string, htmlContent string, params []et.Json, tp TpMessage, createdBy string
* @response et.Items, error
**/
func SendEmail(projectId string, from et.Json, to []et.Json, subject string, htmlContent string, params et.Json, tp TpMessage, createdBy string) (et.Items, error) {
	serviceId := New(projectId, "email", "Send email message", createdBy)
	result, err := brevo.SendEmail(from, to, subject, htmlContent, params, tp.String())
	if err != nil {
		SetStatus(serviceId, STATUS_ERROR, et.Json{
			"error": err.Error(),
		})
		return et.Items{}, err
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
			"type":        tp.String(),
			"createdBy":   createdBy,
			"sender":      "Brevo",
			"result":      result,
		},
	})

	return result, nil
}
