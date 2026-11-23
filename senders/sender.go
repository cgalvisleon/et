package senders

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
)

// Los tipos de mensaje: el kind del cuerpo
const (
	KindNotification = "notification"
	KindCommercial   = "comercial"
)

// SMS: un mensaje de texto a un celular (To con el indicativo del país)
type SMS struct {
	To   string
	Text string
	Kind string
}

// Email: un correo con su asunto, su contenido en html y sus adjuntos
type Email struct {
	From        string
	To          string
	Subject     string
	Html        string
	Kind        string
	Attachments []Attachment
}

// Attachment: un archivo adjunto de un correo
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// WhatsApp: un mensaje de WhatsApp de un tipo (text, template, image, document, audio, video o interactive). Otp es el
// código de una plantilla AUTHENTICATION: cada proveedor lo pone en el cuerpo y en el botón de copiar (Send.Otp)
type WhatsApp struct {
	From     string
	To       string
	Type     string
	Text     string
	Template et.Json // template: {templateName, templateData, language}
	Media    et.Json // image, document, audio, video: {mediaUrl, caption?, filename?}; interactive: {body, action…}
	Metadata et.Json // vuelve en los avisos de entrega del proveedor (callbackData)
	Kind     string
	Otp      string
}

// Sender: un proveedor de envíos con su configuración; cada uno vive en este paquete (infobip.go, meta.go…)
type Sender interface {
	SendSMS(message SMS) (et.Json, error)
	SendEmail(message Email) (et.Json, error)
	SendWhatsApp(message WhatsApp) (et.Json, error)
}

// Los proveedores por su id (el del catálogo sender_provider): cada uno se crea con sus parámetros (los config.fields de
// su kind: los de un tenant en sender_tenants, o los del entorno en envOf)
var providers = map[string]func(params et.Json) (Sender, error){
	"infobip":    NewInfobip,
	"labsmobile": NewLabsMobile,
	"meta":       NewMeta,
	"mock":       NewMock,
	"resend":     NewResend,
}

/**
* New: Un proveedor con sus parámetros
* @param provider string, params et.Json
* @return Sender, error
**/
func New(provider string, params et.Json) (Sender, error) {
	create, ok := providers[provider]
	if !ok {
		return nil, fmt.Errorf(MSG_PROVIDER_NOT_FOUND, provider)
	}

	return create(params)
}
