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
* @param tenantId string, contactNumbers []string, content string, params et.Json, tp TpMessage, createdBy string
* @response et.Item, error
**/
func SendSms(tenantId string, contactNumbers []string, content string, params et.Json, tp TpMessage, createdBy string) (et.Items, error) {
	serviceId := New(tenantId, "sms", MSG_SEND_SMS, createdBy)
	result, err := aws.SendSMS(contactNumbers, content, params, tp.String())
	if err != nil {
		SetStatus(serviceId, STATUS_ERROR, et.Json{
			"error": err.Error(),
		})
		return et.Items{}, err
	}

	SetStatus(serviceId, STATUS_SUCCESS, et.Json{
		"context": et.Json{
			"tenantId":       tenantId,
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
* @param tenantId, templateId string, contactNumbers []string, params []et.Json, tp TpMessage, createdBy string
* @response et.Items, error
**/
func SendWhatsapp(tenantId, templateId string, contactNumbers []string, params []et.Json, tp TpMessage, createdBy string) (et.Items, error) {
	serviceId := New(tenantId, "whatsapp", MSG_SEND_WHATSAPP, createdBy)
	result, err := brevo.SendWhatsapp(contactNumbers, templateId, params, tp.String())
	if err != nil {
		SetStatus(serviceId, STATUS_ERROR, et.Json{
			"error": err.Error(),
		})
		return et.Items{}, err
	}

	SetStatus(serviceId, STATUS_SUCCESS, et.Json{
		"context": et.Json{
			"tenantId":       tenantId,
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
* @param tenantId string, from et.Json, to []et.Json, subject string, htmlContent string, params []et.Json, tp TpMessage, createdBy string
* @response et.Items, error
**/
func SendEmail(tenantId string, from et.Json, to []et.Json, subject string, htmlContent string, params et.Json, tp TpMessage, createdBy string) (et.Items, error) {
	serviceId := New(tenantId, "email", MSG_SEND_EMAIL, createdBy)
	result, err := brevo.SendEmail(from, to, subject, htmlContent, params, tp.String())
	if err != nil {
		SetStatus(serviceId, STATUS_ERROR, et.Json{
			"error": err.Error(),
		})
		return et.Items{}, err
	}

	SetStatus(serviceId, STATUS_SUCCESS, et.Json{
		"context": et.Json{
			"tenantId":    tenantId,
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
