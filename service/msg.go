package service

import "github.com/cgalvisleon/et/envar"

var (
	MSG_SEND_SMS_REQUIRED             = "the field contactNumbers is required"
	MSG_SEND_SMS_CONTENT_REQUIRED     = "the field content is required"
	MSG_SEND_SMS_TP_REQUIRED          = "the field tp is required"
	MSG_SEND_SMS_SENDER_REQUIRED      = "the field sender is required"
	MSG_SEND_WHATSAPP_SENDER_REQUIRED = "the field sender is required"
	MSG_SEND_EMAIL_SENDER_REQUIRED    = "the field sender is required"
	MSG_VERIFY_EMAIL                  = "Email verification"
	MSG_SEND_NOT_FOUND                = "sender not found"
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
		MSG_SEND_NOT_FOUND = "Sender no encontrado"
	}
}
