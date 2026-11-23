package senders

import "github.com/cgalvisleon/et/envar"

var (
	MSG_PROVIDER_NOT_FOUND     = "the send provider %s does not exist"
	MSG_INFOBIP_CONFIG         = "infobip needs base_url and api_key in the integration data of the sender in the tenant"
	MSG_INFOBIP_STATUS         = "infobip answered %d: %s"
	MSG_WHATSAPP_TYPE          = "the WhatsApp type %s is not supported"
	MSG_WHATSAPP_FROM_REQUIRED = "the WhatsApp number of the sender is required (whatsapp_from in the integration data of the sender in the tenant)"
	MSG_LABSMOBILE_CONFIG      = "labsmobile needs username and api_token in the integration data of the sender in the tenant"
	MSG_LABSMOBILE_STATUS      = "labsmobile answered %d: %s"
	MSG_LABSMOBILE_FAILED      = "labsmobile did not send the SMS (code %s): %s"
	MSG_CHANNEL_NOT_SUPPORTED  = "the provider %s does not send %s"
	MSG_META_CONFIG            = "meta needs access_token and phone_number_id in the integration data of the sender in the tenant"
	MSG_META_STATUS            = "meta answered %d: %s"
	MSG_META_FAILED            = "meta answered %d: %s (code %s)"
	MSG_META_WABA_REQUIRED     = "meta needs waba_id (the WhatsApp Business account) in the integration data of the sender in the tenant to manage templates"
	MSG_META_WEBHOOK_OBJECT    = "the webhook is not of a WhatsApp Business account: %s"
	MSG_META_METADATA_TOO_LONG = "the WhatsApp metadata has %d characters and meta accepts up to %d (biz_opaque_callback_data)"
	MSG_RESEND_CONFIG          = "resend needs api_key in the integration data of the sender in the tenant"
	MSG_RESEND_FROM            = "resend needs the sender of the email (email_from in the integration data of the sender in the tenant, from a domain verified in Resend)"
	MSG_RESEND_STATUS          = "resend answered %d: %s"
	MSG_RESEND_FAILED          = "resend did not send the email (%s): %s"
)

/**
* init: Traduce los mensajes según LANG
**/
func init() {
	language := envar.Get("LANG", "en")
	switch language {
	case "es":
		{
			MSG_PROVIDER_NOT_FOUND = "el proveedor de envíos %s no existe"
			MSG_INFOBIP_CONFIG = "infobip necesita base_url y api_key en los datos de integración del sender en el tenant"
			MSG_INFOBIP_STATUS = "infobip respondió %d: %s"
			MSG_WHATSAPP_TYPE = "el tipo de WhatsApp %s no está soportado"
			MSG_WHATSAPP_FROM_REQUIRED = "falta el número de WhatsApp del sender (whatsapp_from en los datos de integración del sender en el tenant)"
			MSG_LABSMOBILE_CONFIG = "labsmobile necesita username y api_token en los datos de integración del sender en el tenant"
			MSG_LABSMOBILE_STATUS = "labsmobile respondió %d: %s"
			MSG_LABSMOBILE_FAILED = "labsmobile no envió el SMS (código %s): %s"
			MSG_CHANNEL_NOT_SUPPORTED = "el proveedor %s no envía %s"
			MSG_META_CONFIG = "meta necesita access_token y phone_number_id en los datos de integración del sender en el tenant"
			MSG_META_STATUS = "meta respondió %d: %s"
			MSG_META_FAILED = "meta respondió %d: %s (código %s)"
			MSG_META_WABA_REQUIRED = "meta necesita waba_id (la cuenta de WhatsApp Business) en los datos de integración del sender en el tenant para manejar plantillas"
			MSG_META_WEBHOOK_OBJECT = "el webhook no es de una cuenta de WhatsApp Business: %s"
			MSG_META_METADATA_TOO_LONG = "la metadata del WhatsApp tiene %d caracteres y meta acepta hasta %d (biz_opaque_callback_data)"
			MSG_RESEND_CONFIG = "resend necesita api_key en los datos de integración del sender en el tenant"
			MSG_RESEND_FROM = "resend necesita el remitente del correo (email_from en los datos de integración del sender en el tenant, de un dominio verificado en Resend)"
			MSG_RESEND_STATUS = "resend respondió %d: %s"
			MSG_RESEND_FAILED = "resend no envió el correo (%s): %s"
		}
	}
}
