package senders

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jwsp"
)

// metaUrl, metaVersion: la Graph API de Meta y su versión por defecto (api_version la cambia)
const (
	metaUrl     = jwsp.DefaultBaseURL
	metaVersion = jwsp.DefaultVersion
)

// metaCallbackMax: el largo máximo de biz_opaque_callback_data en caracteres (512, OpenAPI oficial de Meta
// business-messaging-api_v23.0)
const metaCallbackMax = jwsp.CallbackMax

// Meta: WhatsApp por la API Cloud de Meta, sin intermediario (https://developers.facebook.com/docs/whatsapp/cloud-api), sobre
// el cliente de et/jwsp (HTTP, reintentos ante 429/5xx, errores tipados y plantillas). Sus parámetros (config.fields del kind
// meta): access_token (el token del usuario del sistema, secret del tenant), phone_number_id (el id del número que envía),
// waba_id (la cuenta de WhatsApp Business: sus plantillas) y api_version. Recibe los mensajes con la forma de Infobip (la de
// whatsAppOf y los pasos) y los traduce a los tipos de jwsp
type Meta struct {
	Url           string
	Token         string
	PhoneNumberId string
	WabaId        string
	client        *jwsp.Client
}

// metaFailure: un error de Meta con el mensaje traducido de send (MSG_META_FAILED o MSG_META_STATUS); jwsp.AsError lo abre
type metaFailure struct {
	text  string
	cause *jwsp.Error
}

/**
* Error
* @return string
**/
func (s *metaFailure) Error() string {
	return s.text
}

/**
* Unwrap
* @return error
**/
func (s *metaFailure) Unwrap() error {
	return s.cause
}

/**
* NewMeta: El sender de la API Cloud de Meta con sus parámetros (base_url solo para las pruebas)
* @param params et.Json
* @return Sender, error
**/
func NewMeta(params et.Json) (Sender, error) {
	client := jwsp.New(params.TrimString("access_token"), params.TrimString("phone_number_id"))
	client.BaseURL = strings.TrimRight(cmp.Or(params.TrimString("base_url"), metaUrl), "/")
	client.Version = cmp.Or(params.TrimString("api_version"), metaVersion)
	client.WabaId = params.TrimString("waba_id")
	result := &Meta{
		Url:           client.URL(),
		Token:         client.Token,
		PhoneNumberId: client.PhoneNumberId,
		WabaId:        client.WabaId,
		client:        client,
	}
	if result.Token == "" || result.PhoneNumberId == "" {
		return nil, errors.New(MSG_META_CONFIG)
	}

	return result, nil
}

/**
* metaErrorOf: Un error de jwsp con el mensaje de send: MSG_META_FAILED (estado, mensaje para el usuario o técnico, código) o MSG_META_STATUS sin código;
* los demás errores (validación, red) pasan igual
* @param err error
* @return error
**/
func metaErrorOf(err error) error {
	failure, ok := jwsp.AsError(err)
	if !ok {
		return err
	}
	if failure.Code == 0 {
		return &metaFailure{text: fmt.Sprintf(MSG_META_STATUS, failure.Status, cmp.Or(failure.Body, failure.Message)), cause: failure}
	}
	return &metaFailure{text: fmt.Sprintf(MSG_META_FAILED, failure.Status, failure.Text(), strconv.Itoa(failure.Code)), cause: failure}
}

/**
* SubscribeApp: Suscribe la app de Meta del token a la cuenta de WhatsApp Business (POST /{waba_id}/subscribed_apps sin parámetros: los avisos van a la URL
* de callback de la app; verificado en la documentación de Meta, «Webhooks override»). Meta responde {success: true}
* @return et.Json, error
**/
func (s *Meta) SubscribeApp() (et.Json, error) {
	if s.WabaId == "" {
		return et.Json{}, errors.New(MSG_META_WABA_REQUIRED)
	}

	output, err := s.client.SubscribeApp(context.Background())
	if err != nil {
		return et.Json{}, metaErrorOf(err)
	}
	return output, nil
}

/**
* SendWhatsApp: POST /{phone_number_id}/messages según el tipo; metadata va en biz_opaque_callback_data y falla si pasa de metaCallbackMax
* @param message WhatsApp
* @return et.Json, error
**/
func (s *Meta) SendWhatsApp(message WhatsApp) (et.Json, error) {
	callback, err := metaCallbackOf(message.Metadata)
	if err != nil {
		return et.Json{}, err
	}

	send := jwsp.Message{To: message.To, CallbackData: callback}
	switch message.Type {
	case "text":
		send.Text = &jwsp.Text{Body: message.Text}
	case "image", "document", "audio", "video":
		media := &jwsp.Media{Link: message.Media.Str("mediaUrl")}
		if message.Type != "audio" {
			media.Caption = message.Media.Str("caption")
		}
		if message.Type == "document" {
			media.Filename = message.Media.Str("filename")
		}
		switch message.Type {
		case "image":
			send.Image = media
		case "document":
			send.Document = media
		case "audio":
			send.Audio = media
		default:
			send.Video = media
		}
	case "interactive":
		send.Interactive = metaInteractiveOf(message.Media)
	case "template":
		send.Template = metaTemplateOf(message.Template, message.Otp)
	default:
		return et.Json{}, fmt.Errorf(MSG_WHATSAPP_TYPE, message.Type)
	}

	result, err := s.client.Send(context.Background(), send)
	if err != nil {
		return et.Json{}, metaErrorOf(err)
	}

	return et.Json{"provider": "meta", "response": result.Raw}, nil
}

/**
* metaCallbackOf: biz_opaque_callback_data desde metadata (texto JSON; vacío sin metadata); más de metaCallbackMax caracteres → MSG_META_METADATA_TOO_LONG
* @param metadata et.Json
* @return string, error
**/
func metaCallbackOf(metadata et.Json) (string, error) {
	if len(metadata) == 0 {
		return "", nil
	}

	result := metadata.ToString()
	if size := utf8.RuneCountInString(result); size > metaCallbackMax {
		return "", fmt.Errorf(MSG_META_METADATA_TOO_LONG, size, metaCallbackMax)
	}
	return result, nil
}

/**
* metaInteractiveOf: Los botones de respuesta de jwsp desde la forma de Infobip ({body: {text}, header?: {type, text}, footer?: {text}, action: {buttons:
* [{type, id, title}]}})
* @param media et.Json
* @return *jwsp.Interactive
**/
func metaInteractiveOf(media et.Json) *jwsp.Interactive {
	buttons := []jwsp.ReplyButton{}
	for _, button := range media.Json("action").ArrayJson("buttons") {
		buttons = append(buttons, jwsp.ReplyButton{Type: "reply", Reply: jwsp.Button{Id: button.Str("id"), Title: button.Str("title")}})
	}

	result := &jwsp.Interactive{
		Type:   "button",
		Body:   &jwsp.TextBody{Text: media.Json("body").Str("text")},
		Action: jwsp.Action{Buttons: buttons},
	}
	if header := media.Json("header").Str("text"); header != "" {
		result.Header = &jwsp.InteractiveHeader{Type: "text", Text: header}
	}
	if footer := media.Json("footer").Str("text"); footer != "" {
		result.Footer = &jwsp.TextBody{Text: footer}
	}
	return result
}

/**
* metaTemplateOf: Una plantilla aprobada de jwsp desde la forma de Infobip; con otp, el código va en el cuerpo y en el botón de copiar (jwsp.OtpComponents)
* @param template et.Json, otp string
* @return *jwsp.Template
**/
func metaTemplateOf(template et.Json, otp string) *jwsp.Template {
	result := &jwsp.Template{
		Name:     template.Str("templateName"),
		Language: cmp.Or(template.Str("language"), "es"),
	}
	if otp != "" {
		result.Components = jwsp.OtpComponents(otp)
		return result
	}

	data := template.Json("templateData")
	if header := data.Json("header").Str("placeholder"); header != "" {
		result.Components = append(result.Components, jwsp.HeaderComponent(header))
	}
	if values := data.Json("body").ArrayStr("placeholders"); len(values) > 0 {
		result.Components = append(result.Components, jwsp.BodyComponent(values...))
	}
	return result
}

/**
* SendSMS: Meta no envía SMS
* @param message SMS
* @return et.Json, error
**/
func (s *Meta) SendSMS(message SMS) (et.Json, error) {
	return et.Json{}, fmt.Errorf(MSG_CHANNEL_NOT_SUPPORTED, "meta", "sms")
}

/**
* SendEmail: Meta no envía correos
* @param message Email
* @return et.Json, error
**/
func (s *Meta) SendEmail(message Email) (et.Json, error) {
	return et.Json{}, fmt.Errorf(MSG_CHANNEL_NOT_SUPPORTED, "meta", "email")
}

/**
* metaSpecOf: Los componentes de una plantilla de jwsp desde la estructura de Infobip ({header?: {format, text, example?}, body: {text, examples?},
* footer?: {text}, buttons?: [{type, text}]}); una estructura AUTHENTICATION (un botón con otpType) con jwsp.AuthComponents
* @param structure et.Json
* @return []jwsp.TemplateComponent
**/
func metaSpecOf(structure et.Json) []jwsp.TemplateComponent {
	if isAuthStructure(structure) {
		return metaAuthComponentsOf(structure)
	}

	components := []jwsp.TemplateComponent{}
	if header := structure.Json("header"); header.Str("text") != "" {
		component := jwsp.TemplateComponent{Type: "HEADER", Format: cmp.Or(header.Str("format"), "TEXT"), Text: header.Str("text")}
		if example := header.Str("example"); example != "" {
			component.Example = &jwsp.TemplateExample{HeaderText: []string{example}}
		}
		components = append(components, component)
	}

	body := structure.Json("body")
	component := jwsp.TemplateComponent{Type: "BODY", Text: body.Str("text")}
	if examples := body.ArrayStr("examples"); len(examples) > 0 {
		component.Example = &jwsp.TemplateExample{BodyText: [][]string{examples}}
	}
	components = append(components, component)

	if footer := structure.Json("footer").Str("text"); footer != "" {
		components = append(components, jwsp.TemplateComponent{Type: "FOOTER", Text: footer})
	}

	buttons := []jwsp.TemplateButton{}
	for _, button := range structure.ArrayJson("buttons") {
		buttons = append(buttons, jwsp.TemplateButton{Type: cmp.Or(button.Str("type"), "QUICK_REPLY"), Text: button.Str("text")})
	}
	if len(buttons) > 0 {
		components = append(components, jwsp.TemplateComponent{Type: "BUTTONS", Buttons: buttons})
	}
	return components
}

/**
* metaComponentsOf: Los componentes de metaSpecOf como van a Meta (JSON)
* @param structure et.Json
* @return []et.Json
**/
func metaComponentsOf(structure et.Json) []et.Json {
	result := []et.Json{}
	raw, err := json.Marshal(metaSpecOf(structure))
	if err != nil {
		return result
	}
	json.Unmarshal(raw, &result)
	return result
}

/**
* isAuthStructure: Si la estructura es la de una plantilla AUTHENTICATION (un botón con otpType)
* @param structure et.Json
* @return bool
**/
func isAuthStructure(structure et.Json) bool {
	for _, button := range structure.ArrayJson("buttons") {
		if button.Str("otpType") != "" {
			return true
		}
	}
	return false
}

/**
* metaAuthComponentsOf: Los componentes de una plantilla AUTHENTICATION desde la estructura de Infobip (el primer botón con otpType; Meta admite uno)
* @param structure et.Json
* @return []jwsp.TemplateComponent
**/
func metaAuthComponentsOf(structure et.Json) []jwsp.TemplateComponent {
	security := structure.Json("body").Bool("addSecurityRecommendation")
	minutes := structure.Json("footer").Int("codeExpirationMinutes")
	otpType, text := "", ""
	for _, button := range structure.ArrayJson("buttons") {
		if button.Str("otpType") != "" {
			otpType, text = button.Str("otpType"), button.Str("text")
			break
		}
	}
	return jwsp.AuthComponents(security, minutes, otpType, text)
}

/**
* metaQualityOf: La calidad de una plantilla de Meta (GREEN, YELLOW, RED) como la de Infobip (HIGH, MEDIUM, LOW; sin ella, UNKNOWN)
* @param score string
* @return string
**/
func metaQualityOf(score string) string {
	switch score {
	case "GREEN":
		return "HIGH"
	case "YELLOW":
		return "MEDIUM"
	case "RED":
		return "LOW"
	}
	return "UNKNOWN"
}

/**
* PublishWhatsAppTemplate: Crea la plantilla en la cuenta (waba_id); con el id de Meta la edita (componentes y, si viene, categoría) y lee su estado.
* sender no se usa: las plantillas son de la cuenta
* @param sender string, template WhatsAppTemplate
* @return TemplateState, error
**/
func (s *Meta) PublishWhatsAppTemplate(sender string, template WhatsAppTemplate) (TemplateState, error) {
	if s.WabaId == "" {
		return TemplateState{}, errors.New(MSG_META_WABA_REQUIRED)
	}

	ctx := context.Background()
	components := metaSpecOf(template.Structure)
	if template.Id != "" {
		update := jwsp.TemplateUpdate{Category: template.Category, Components: components}
		if err := s.client.UpdateTemplate(ctx, template.Id, update); err != nil {
			return TemplateState{}, metaErrorOf(err)
		}

		state, found, err := s.WhatsAppTemplateState(sender, template.Name, template.Language)
		if err != nil || found {
			return state, err
		}
		return stateOf(et.Json{}, template), nil
	}

	spec := jwsp.TemplateSpec{Name: template.Name, Language: template.Language, Category: template.Category, Components: components}
	created, err := s.client.CreateTemplate(ctx, spec)
	if err != nil {
		return TemplateState{}, metaErrorOf(err)
	}

	output := et.Json{"id": created.Id, "status": created.Status, "category": created.Category}
	return stateOf(output, template), nil
}

/**
* WhatsAppTemplateState: La plantilla con el mismo nombre e idioma (Meta filtra por nombre parecido; jwsp recorre las páginas)
* @param sender, name, language string
* @return TemplateState, bool, error
**/
func (s *Meta) WhatsAppTemplateState(sender, name, language string) (TemplateState, bool, error) {
	if s.WabaId == "" {
		return TemplateState{}, false, errors.New(MSG_META_WABA_REQUIRED)
	}

	item, found, err := s.client.Template(context.Background(), name, language)
	if err != nil {
		return TemplateState{}, false, metaErrorOf(err)
	}
	if !found {
		return TemplateState{}, false, nil
	}

	output := et.Json{"id": item.Id, "name": item.Name, "language": item.Language, "status": item.Status, "category": item.Category, "quality": metaQualityOf(item.Quality)}
	return stateOf(output, WhatsAppTemplate{Name: name, Language: language}), true, nil
}

/**
* DeleteWhatsAppTemplate: Borra todas las plantillas de ese nombre, en cada idioma
* @param sender, name string
* @return error
**/
func (s *Meta) DeleteWhatsAppTemplate(sender, name string) error {
	if s.WabaId == "" {
		return errors.New(MSG_META_WABA_REQUIRED)
	}
	return metaErrorOf(s.client.DeleteTemplate(context.Background(), name, ""))
}
