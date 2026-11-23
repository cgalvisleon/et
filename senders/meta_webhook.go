package senders

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/jwsp"
)

// El lector del webhook de la API Cloud de Meta (campo messages), sobre et/jwsp desde F-17 (2026-11-21): estos nombres son
// los de F-18 apuntando a jwsp, así send/meta_hooks.go no cambia. Formatos verificados el 2026-11-16 en la documentación de
// Meta: «Create a webhook endpoint» (GET con hub.mode, hub.challenge y hub.verify_token; POST con X-Hub-Signature-256:
// sha256=<HMAC-SHA256 del cuerpo con el app secret>) y las referencias de webhooks de mensajes (text, button, interactive,
// status y errors)

// Los tipos de evento del webhook
const (
	MetaEventMessage = jwsp.EventMessage
	MetaEventStatus  = jwsp.EventStatus
	MetaEventError   = jwsp.EventError
)

// MetaError: un error de Meta en el webhook (value.errors, messages[].errors o statuses[].errors)
type MetaError = jwsp.WebhookError

// MetaReply: la respuesta a un mensaje con botones (button, button_reply o list_reply)
type MetaReply = jwsp.Reply

// MetaMessage: un mensaje que un usuario de WhatsApp le escribió al número
type MetaMessage = jwsp.IncomingMessage

// MetaStatus: el estado de un mensaje que envió el número; CallbackData es el biz_opaque_callback_data con que se envió
type MetaStatus = jwsp.Status

// MetaEvent: un evento del webhook con el número (phone_number_id) y la cuenta (waba_id) a los que llegó
type MetaEvent = jwsp.Event

/**
* MetaSignatureOk: Si X-Hub-Signature-256 (sha256=<hex>) es el HMAC-SHA256 del cuerpo crudo con el app secret (jwsp.VerifySignature, tiempo constante).
* Sin firma o sin app secret, false
* @param raw []byte, signature, appSecret string
* @return bool
**/
func MetaSignatureOk(raw []byte, signature, appSecret string) bool {
	return jwsp.VerifySignature(raw, signature, appSecret)
}

/**
* MetaSignature: La firma de un cuerpo como la manda Meta (sha256=<hex>); para las pruebas
* @param raw []byte, appSecret string
* @return string
**/
func MetaSignature(raw []byte, appSecret string) string {
	return jwsp.Signature(raw, appSecret)
}

/**
* MetaVerifyTokenOk: Si el hub.verify_token de la suscripción es el esperado (tiempo constante; vacío nunca vale)
* @param got, want string
* @return bool
**/
func MetaVerifyTokenOk(got, want string) bool {
	return jwsp.VerifyTokenOk(got, want)
}

/**
* ParseMetaWebhook: Los eventos de un webhook de Meta (jwsp.ParseEvents); otro object → MSG_META_WEBHOOK_OBJECT. No verifica la firma: va antes, con
* MetaSignatureOk (el app secret depende del número o del hook)
* @param raw []byte
* @return []MetaEvent, error
**/
func ParseMetaWebhook(raw []byte) ([]MetaEvent, error) {
	var head struct {
		Object string `json:"object"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	if head.Object != "whatsapp_business_account" {
		return nil, fmt.Errorf(MSG_META_WEBHOOK_OBJECT, head.Object)
	}

	return jwsp.ParseEvents(raw)
}
