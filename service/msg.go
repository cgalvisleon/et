package service

import "github.com/cgalvisleon/et/envar"

var (
	MSG_SEND_SMS_REQUIRED             = "El campo contactNumbers es requerido"
	MSG_SEND_SMS_CONTENT_REQUIRED     = "El campo content es requerido"
	MSG_SEND_SMS_TP_REQUIRED          = "El campo tp es requerido"
	MSG_SEND_SMS_SENDER_REQUIRED      = "El campo sender es requerido"
	MSG_SEND_WHATSAPP_SENDER_REQUIRED = "El campo sender es requerido"
	MSG_SEND_EMAIL_SENDER_REQUIRED    = "El campo sender es requerido"
	MSG_VERIFY_EMAIL                  = "Verificación de email"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_SEND_SMS_REQUIRED = "El campo contactNumbers es requerido"
		MSG_SEND_SMS_CONTENT_REQUIRED = "El campo content es requerido"
		MSG_SEND_SMS_TP_REQUIRED = "El campo tp es requerido"
		MSG_SEND_SMS_SENDER_REQUIRED = "El campo sender es requerido"
		MSG_SEND_WHATSAPP_SENDER_REQUIRED = "El campo sender es requerido"
		MSG_SEND_EMAIL_SENDER_REQUIRED = "El campo sender es requerido"
		MSG_VERIFY_EMAIL = "Verificación de email"
	}
}
