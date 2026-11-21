package jwsp

import (
	"errors"

	"github.com/cgalvisleon/et/envar"
)

// Los mensajes del paquete (en inglés; español con LANG=es)
var (
	MSG_TOKEN_REQUIRED     = "jwsp: the access token is required"
	MSG_PHONE_REQUIRED     = "jwsp: the phone_number_id is required"
	MSG_WABA_REQUIRED      = "jwsp: the waba_id (WhatsApp Business account) is required"
	MSG_TO_REQUIRED        = "jwsp: the recipient (to) is required"
	MSG_MESSAGE_EMPTY      = "jwsp: the message has no content"
	MSG_MESSAGE_AMBIGUOUS  = "jwsp: the message has more than one content: %s"
	MSG_CALLBACK_TOO_LONG  = "jwsp: callback_data has %d characters and meta accepts up to %d"
	MSG_SIGNATURE          = "jwsp: invalid webhook signature"
	MSG_BODY_TOO_LARGE     = "jwsp: the webhook body is too large"
	MSG_WEBHOOK_OBJECT     = "jwsp: the webhook is not of a WhatsApp Business account: %s"
	MSG_META_ERROR         = "meta answered %d: %s (code %d)"
	MSG_META_STATUS        = "meta answered %d: %s"
	MSG_MEDIA_ID_REQUIRED  = "jwsp: the media id is required"
	MSG_TEMPLATE_REQUIRED  = "jwsp: the template name or id is required"
	MSG_MEDIA_URL_REQUIRED = "jwsp: the media url is required"
)

// ErrSignature, ErrBodyTooLarge: los errores de ParseWebhook que un handler traduce a 401 y 413
var (
	ErrSignature    error
	ErrBodyTooLarge error
)

/**
* init
**/
func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_TOKEN_REQUIRED = "jwsp: el token de acceso es requerido"
		MSG_PHONE_REQUIRED = "jwsp: el phone_number_id es requerido"
		MSG_WABA_REQUIRED = "jwsp: el waba_id (la cuenta de WhatsApp Business) es requerido"
		MSG_TO_REQUIRED = "jwsp: el destinatario (to) es requerido"
		MSG_MESSAGE_EMPTY = "jwsp: el mensaje no tiene contenido"
		MSG_MESSAGE_AMBIGUOUS = "jwsp: el mensaje tiene más de un contenido: %s"
		MSG_CALLBACK_TOO_LONG = "jwsp: callback_data tiene %d caracteres y meta acepta hasta %d"
		MSG_SIGNATURE = "jwsp: la firma del webhook no es válida"
		MSG_BODY_TOO_LARGE = "jwsp: el cuerpo del webhook es demasiado grande"
		MSG_WEBHOOK_OBJECT = "jwsp: el webhook no es de una cuenta de WhatsApp Business: %s"
		MSG_META_ERROR = "meta respondió %d: %s (código %d)"
		MSG_META_STATUS = "meta respondió %d: %s"
		MSG_MEDIA_ID_REQUIRED = "jwsp: el id del medio es requerido"
		MSG_TEMPLATE_REQUIRED = "jwsp: el nombre o el id de la plantilla es requerido"
		MSG_MEDIA_URL_REQUIRED = "jwsp: la url del medio es requerida"
	}

	ErrSignature = errors.New(MSG_SIGNATURE)
	ErrBodyTooLarge = errors.New(MSG_BODY_TOO_LARGE)
}
