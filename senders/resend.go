package senders

import (
	"cmp"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
)

// resendUrl: el envío de correos de Resend (https://resend.com/docs/api-reference/emails/send-email)
const resendUrl = "https://api.resend.com/emails"

// Resend: un proveedor solo de correo. Sus parámetros (config.fields del kind resend): api_key (re_…, secret del tenant)
// y email_from (el remitente por defecto, de un dominio verificado en Resend: «Nombre <no-reply@dominio.com>»)
type Resend struct {
	Url       string
	ApiKey    string
	EmailFrom string
}

/**
* NewResend: El sender de Resend con sus parámetros (base_url solo para las pruebas)
* @param params et.Json
* @return Sender, error
**/
func NewResend(params et.Json) (Sender, error) {
	result := &Resend{
		Url:       cmp.Or(params.TrimString("base_url"), resendUrl),
		ApiKey:    params.TrimString("api_key"),
		EmailFrom: params.TrimString("email_from"),
	}
	if result.ApiKey == "" {
		return nil, errors.New(MSG_RESEND_CONFIG)
	}

	return result, nil
}

/**
* SendEmail: POST /emails con el token Bearer: remitente (el del correo o email_from), destinatario, asunto, HTML y los adjuntos en base64. Resend
* responde {id} si lo aceptó; si no, {statusCode, name, message}
* @param message Email
* @return et.Json, error
**/
func (s *Resend) SendEmail(message Email) (et.Json, error) {
	from := cmp.Or(message.From, s.EmailFrom)
	if from == "" {
		return et.Json{}, errors.New(MSG_RESEND_FROM)
	}

	body := et.Json{
		"from":    from,
		"to":      []string{message.To},
		"subject": message.Subject,
		"html":    message.Html,
	}
	attachments := []et.Json{}
	for _, file := range message.Attachments {
		attachment := et.Json{"filename": file.Filename, "content": base64.StdEncoding.EncodeToString(file.Content)}
		if file.ContentType != "" {
			attachment["content_type"] = file.ContentType
		}
		attachments = append(attachments, attachment)
	}
	if len(attachments) > 0 {
		body["attachments"] = attachments
	}

	header := et.Json{
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + s.ApiKey,
	}
	res, status := request.Fetch(http.MethodPost, s.Url, header, body, emailTimeout, nil)
	if res == nil || len(res.Data) == 0 {
		return et.Json{}, fmt.Errorf(MSG_RESEND_STATUS, status.Code, status.Message)
	}

	output, err := res.ToJson()
	if err != nil {
		return et.Json{}, fmt.Errorf(MSG_RESEND_STATUS, status.Code, res.ToString())
	}

	if output.Str("id") == "" {
		return et.Json{}, fmt.Errorf(MSG_RESEND_FAILED, cmp.Or(output.Str("name"), fmt.Sprint(status.Code)), output.Str("message"))
	}

	return et.Json{"provider": "resend", "response": output}, nil
}

/**
* SendSMS: Resend no envía SMS
* @param message SMS
* @return et.Json, error
**/
func (s *Resend) SendSMS(message SMS) (et.Json, error) {
	return et.Json{}, fmt.Errorf(MSG_CHANNEL_NOT_SUPPORTED, "resend", "sms")
}

/**
* SendWhatsApp: Resend no envía WhatsApp
* @param message WhatsApp
* @return et.Json, error
**/
func (s *Resend) SendWhatsApp(message WhatsApp) (et.Json, error) {
	return et.Json{}, fmt.Errorf(MSG_CHANNEL_NOT_SUPPORTED, "resend", "whatsapp")
}
