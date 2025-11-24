package service

import (
	"fmt"

	"github.com/cgalvisleon/et/aws"
	"github.com/cgalvisleon/et/brevo"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
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
func SendSms(tenantId, serviceId string, contactNumbers []string, content string, params et.Json, tp TpMessage, createdBy string) (et.Items, error) {
	result, err := aws.SendSMS(contactNumbers, content, params, tp.String())
	if err != nil {
		return et.Items{}, err
	}

	serviceId = reg.TagULID("service", serviceId)
	if set != nil {
		set(serviceId, et.Json{
			"tenantId":       tenantId,
			"serviceId":      serviceId,
			"service":        SERVICE_SMS,
			"contactNumbers": contactNumbers,
			"content":        content,
			"params":         params,
			"type":           tp.String(),
			"createdBy":      createdBy,
			"sender":         "AWS SNS",
			"result":         result,
		})
	}

	return result, nil
}

/**
* SendWhatsapp
* @param tenantId, serviceId, templateId string, contactNumbers []string, params []et.Json, tp TpMessage, createdBy string
* @response et.Items, error
**/
func SendWhatsapp(tenantId, serviceId, templateId string, contactNumbers []string, params []et.Json, tp TpMessage, createdBy string) (et.Items, error) {
	result, err := brevo.SendWhatsapp(contactNumbers, templateId, params, tp.String())
	if err != nil {
		return et.Items{}, err
	}

	serviceId = reg.TagULID("service", serviceId)
	if set != nil {
		set(serviceId, et.Json{
			"tenantId":       tenantId,
			"serviceId":      serviceId,
			"service":        SERVICE_WHATSAPP,
			"templateId":     templateId,
			"contactNumbers": contactNumbers,
			"params":         params,
			"type":           tp.String(),
			"createdBy":      createdBy,
			"sender":         "Brevo",
			"result":         result,
		})
	}

	return result, nil
}

/**
* SendEmail
* @param tenantId, serviceId string, from et.Json, to []et.Json, subject string, htmlContent string, params []et.Json, tp TpMessage, createdBy string
* @response et.Items, error
**/
func SendEmail(tenantId, serviceId string, from et.Json, to []et.Json, subject string, htmlContent string, params et.Json, tp TpMessage, createdBy string) (et.Items, error) {
	result, err := brevo.SendEmail(from, to, subject, htmlContent, params, tp.String())
	if err != nil {
		return et.Items{}, err
	}

	serviceId = reg.TagULID("service", serviceId)
	if set != nil {
		set(serviceId, et.Json{
			"tenantId":    tenantId,
			"serviceId":   serviceId,
			"service":     SERVICE_EMAIL,
			"from":        from,
			"to":          to,
			"subject":     subject,
			"htmlContent": htmlContent,
			"params":      params,
			"type":        tp.String(),
			"createdBy":   createdBy,
			"sender":      "Brevo",
			"result":      result,
		})
	}

	return result, nil
}

/**
* SendEmailByTemplateId
* @param tenantId, serviceId string, from et.Json, to []et.Json, subject string, templateId string, params et.Json, tp TpMessage, createdBy string
* @response et.Items, error
**/
func SendEmailByTemplateId(tenantId, serviceId string, from et.Json, to []et.Json, subject string, templateId string, params et.Json, tp TpMessage, createdBy string) (et.Items, error) {
	if getTemplate == nil {
		return et.Items{}, fmt.Errorf("get template is nil")
	}

	template, err := getTemplate(templateId)
	if err != nil {
		return et.Items{}, err
	}

	return SendEmail(tenantId, serviceId, from, to, subject, template, params, tp, createdBy)
}
