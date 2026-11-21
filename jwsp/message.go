package jwsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cgalvisleon/et/et"
)

// CallbackMax: el largo máximo de biz_opaque_callback_data en caracteres (OpenAPI oficial de Meta, maxLength 512)
const CallbackMax = 512

// Message: un mensaje saliente con un solo contenido (Text, una media, Template, Interactive, Location, Contacts o Reaction).
// ReplyTo es el id (wamid) del mensaje al que responde (context); CallbackData vuelve en los estados del webhook
type Message struct {
	To           string
	ReplyTo      string
	CallbackData string
	Text         *Text
	Image        *Media
	Video        *Media
	Audio        *Media
	Document     *Media
	Sticker      *Media
	Template     *Template
	Interactive  *Interactive
	Location     *Location
	Contacts     []Contact
	Reaction     *Reaction
}

// Text: un texto; PreviewURL muestra la vista previa del primer enlace
type Text struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url,omitempty"`
}

// Media: una imagen, video, audio, documento o sticker por su id (subido con UploadMedia) o por un link público. Caption no
// aplica a audio ni sticker; Filename solo a documentos
type Media struct {
	Id       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// Location: una ubicación
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

// Reaction: una reacción (emoji) a un mensaje; Emoji vacío la quita
type Reaction struct {
	MessageId string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

// Contact: una tarjeta de contacto
type Contact struct {
	Name      ContactName      `json:"name"`
	Phones    []ContactPhone   `json:"phones,omitempty"`
	Emails    []ContactEmail   `json:"emails,omitempty"`
	Urls      []ContactUrl     `json:"urls,omitempty"`
	Addresses []ContactAddress `json:"addresses,omitempty"`
	Org       *ContactOrg      `json:"org,omitempty"`
	Birthday  string           `json:"birthday,omitempty"`
}

// ContactName: el nombre de un contacto (FormattedName es obligatorio)
type ContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	MiddleName    string `json:"middle_name,omitempty"`
	Prefix        string `json:"prefix,omitempty"`
	Suffix        string `json:"suffix,omitempty"`
}

// ContactPhone: un teléfono (Type HOME, WORK, CELL…; WaId si tiene WhatsApp)
type ContactPhone struct {
	Phone string `json:"phone"`
	Type  string `json:"type,omitempty"`
	WaId  string `json:"wa_id,omitempty"`
}

// ContactEmail: un correo
type ContactEmail struct {
	Email string `json:"email"`
	Type  string `json:"type,omitempty"`
}

// ContactUrl: una dirección web
type ContactUrl struct {
	Url  string `json:"url"`
	Type string `json:"type,omitempty"`
}

// ContactAddress: una dirección postal
type ContactAddress struct {
	Street      string `json:"street,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	Zip         string `json:"zip,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Type        string `json:"type,omitempty"`
}

// ContactOrg: la empresa de un contacto
type ContactOrg struct {
	Company    string `json:"company,omitempty"`
	Department string `json:"department,omitempty"`
	Title      string `json:"title,omitempty"`
}

// Template: una plantilla aprobada para enviar (Language es el código, p. ej. es o en_US)
type Template struct {
	Name       string      `json:"name"`
	Language   string      `json:"-"`
	Components []Component `json:"components,omitempty"`
}

/**
* MarshalJSON: language va como {code}
* @return []byte, error
**/
func (s Template) MarshalJSON() ([]byte, error) {
	type plain Template
	return json.Marshal(struct {
		plain
		Language et.Json `json:"language"`
	}{plain(s), et.Json{"code": s.Language}})
}

// Component: los parámetros de un componente de plantilla al enviar: header, body o button (con SubType url, quick_reply,
// copy_code o flow e Index, la posición del botón desde 0)
type Component struct {
	Type       string      `json:"type"`
	SubType    string      `json:"sub_type,omitempty"`
	Index      string      `json:"index,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty"`
}

// Parameter: un parámetro de plantilla: text, currency, date_time, image, video, document, payload o coupon_code. Name es
// el parameter_name de las plantillas con parámetros con nombre
type Parameter struct {
	Type       string    `json:"type"`
	Name       string    `json:"parameter_name,omitempty"`
	Text       string    `json:"text,omitempty"`
	Payload    string    `json:"payload,omitempty"`
	CouponCode string    `json:"coupon_code,omitempty"`
	Currency   *Currency `json:"currency,omitempty"`
	DateTime   *DateTime `json:"date_time,omitempty"`
	Image      *Media    `json:"image,omitempty"`
	Video      *Media    `json:"video,omitempty"`
	Document   *Media    `json:"document,omitempty"`
}

// Currency: un monto (Amount1000 es el monto por 1000)
type Currency struct {
	FallbackValue string `json:"fallback_value"`
	Code          string `json:"code"`
	Amount1000    int64  `json:"amount_1000"`
}

// DateTime: una fecha como texto de respaldo
type DateTime struct {
	FallbackValue string `json:"fallback_value"`
}

// Interactive: un mensaje interactivo: button (hasta 3 respuestas), list (secciones con filas) o cta_url (un botón con enlace)
type Interactive struct {
	Type   string             `json:"type"`
	Header *InteractiveHeader `json:"header,omitempty"`
	Body   *TextBody          `json:"body,omitempty"`
	Footer *TextBody          `json:"footer,omitempty"`
	Action Action             `json:"action"`
}

// InteractiveHeader: el encabezado: text, image, video o document
type InteractiveHeader struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Image    *Media `json:"image,omitempty"`
	Video    *Media `json:"video,omitempty"`
	Document *Media `json:"document,omitempty"`
}

// TextBody: un texto de cuerpo o pie
type TextBody struct {
	Text string `json:"text"`
}

// Action: los botones de respuesta (Buttons), el botón y las secciones de una lista (Button, Sections) o un cta_url (Name y
// Parameters {display_text, url})
type Action struct {
	Buttons    []ReplyButton `json:"buttons,omitempty"`
	Button     string        `json:"button,omitempty"`
	Sections   []Section     `json:"sections,omitempty"`
	Name       string        `json:"name,omitempty"`
	Parameters et.Json       `json:"parameters,omitempty"`
}

// ReplyButton: un botón de respuesta (Type reply)
type ReplyButton struct {
	Type  string `json:"type"`
	Reply Button `json:"reply"`
}

// Button: el id que vuelve en el webhook y el título del botón
type Button struct {
	Id    string `json:"id"`
	Title string `json:"title"`
}

// Section: una sección de una lista
type Section struct {
	Title string `json:"title,omitempty"`
	Rows  []Row  `json:"rows"`
}

// Row: una fila de una lista
type Row struct {
	Id          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// SendResult: la respuesta de un envío; Raw es lo que respondió Meta
type SendResult struct {
	MessageId     string  `json:"message_id"`
	MessageStatus string  `json:"message_status,omitempty"`
	WaId          string  `json:"wa_id,omitempty"`
	Input         string  `json:"input,omitempty"`
	Raw           et.Json `json:"raw"`
}

/**
* TextMessage
* @param to, body string
* @return Message
**/
func TextMessage(to, body string) Message {
	return Message{To: to, Text: &Text{Body: body}}
}

/**
* TemplateMessage
* @param to, name, language string, components ...Component
* @return Message
**/
func TemplateMessage(to, name, language string, components ...Component) Message {
	return Message{To: to, Template: &Template{Name: name, Language: language, Components: components}}
}

/**
* ButtonsMessage: hasta 3 botones de respuesta
* @param to, body string, buttons ...Button
* @return Message
**/
func ButtonsMessage(to, body string, buttons ...Button) Message {
	items := []ReplyButton{}
	for _, button := range buttons {
		items = append(items, ReplyButton{Type: "reply", Reply: button})
	}
	interactive := &Interactive{Type: "button", Body: &TextBody{Text: body}, Action: Action{Buttons: items}}
	return Message{To: to, Interactive: interactive}
}

/**
* ListMessage: una lista con el texto de su botón y sus secciones
* @param to, body, button string, sections ...Section
* @return Message
**/
func ListMessage(to, body, button string, sections ...Section) Message {
	interactive := &Interactive{Type: "list", Body: &TextBody{Text: body}, Action: Action{Button: button, Sections: sections}}
	return Message{To: to, Interactive: interactive}
}

/**
* UrlButtonMessage: un botón con enlace (cta_url)
* @param to, body, text, link string
* @return Message
**/
func UrlButtonMessage(to, body, text, link string) Message {
	action := Action{Name: "cta_url", Parameters: et.Json{"display_text": text, "url": link}}
	return Message{To: to, Interactive: &Interactive{Type: "cta_url", Body: &TextBody{Text: body}, Action: action}}
}

/**
* TextParams: parámetros de texto en orden
* @param values ...string
* @return []Parameter
**/
func TextParams(values ...string) []Parameter {
	result := []Parameter{}
	for _, value := range values {
		result = append(result, Parameter{Type: "text", Text: value})
	}
	return result
}

/**
* BodyComponent: el cuerpo con sus parámetros de texto
* @param values ...string
* @return Component
**/
func BodyComponent(values ...string) Component {
	return Component{Type: "body", Parameters: TextParams(values...)}
}

/**
* HeaderComponent: el encabezado con un texto
* @param value string
* @return Component
**/
func HeaderComponent(value string) Component {
	return Component{Type: "header", Parameters: TextParams(value)}
}

/**
* HeaderMediaComponent: el encabezado con una imagen, video o documento (kind image|video|document)
* @param kind string, media Media
* @return Component
**/
func HeaderMediaComponent(kind string, media Media) Component {
	parameter := Parameter{Type: kind}
	switch kind {
	case "video":
		parameter.Video = &media
	case "document":
		parameter.Document = &media
	default:
		parameter.Type = "image"
		parameter.Image = &media
	}
	return Component{Type: "header", Parameters: []Parameter{parameter}}
}

/**
* UrlButtonComponent: el sufijo dinámico del botón de URL en la posición index
* @param index int, value string
* @return Component
**/
func UrlButtonComponent(index int, value string) Component {
	return Component{Type: "button", SubType: "url", Index: strconv.Itoa(index), Parameters: TextParams(value)}
}

/**
* QuickReplyComponent: el payload que vuelve en el webhook al tocar el botón de respuesta rápida en la posición index
* @param index int, payload string
* @return Component
**/
func QuickReplyComponent(index int, payload string) Component {
	parameters := []Parameter{{Type: "payload", Payload: payload}}
	return Component{Type: "button", SubType: "quick_reply", Index: strconv.Itoa(index), Parameters: parameters}
}

/**
* CopyCodeComponent: el código del botón de copiar código de una plantilla de marketing en la posición index
* @param index int, code string
* @return Component
**/
func CopyCodeComponent(index int, code string) Component {
	parameters := []Parameter{{Type: "coupon_code", CouponCode: code}}
	return Component{Type: "button", SubType: "copy_code", Index: strconv.Itoa(index), Parameters: parameters}
}

/**
* OtpComponents: el código de una plantilla AUTHENTICATION: en el cuerpo y en el botón (sub_type url, index 0), como lo pide Meta
* también para el botón COPY_CODE
* @param code string
* @return []Component
**/
func OtpComponents(code string) []Component {
	return []Component{BodyComponent(code), UrlButtonComponent(0, code)}
}

/**
* body: el cuerpo de POST /{phone_number_id}/messages; valida que haya un solo contenido y el largo de CallbackData
* @return et.Json, error
**/
func (s Message) body() (et.Json, error) {
	if strings.TrimSpace(s.To) == "" {
		return et.Json{}, errors.New(MSG_TO_REQUIRED)
	}
	if size := utf8.RuneCountInString(s.CallbackData); size > CallbackMax {
		return et.Json{}, fmt.Errorf(MSG_CALLBACK_TOO_LONG, size, CallbackMax)
	}

	contents := map[string]any{}
	add := func(kind string, set bool, value any) {
		if set {
			contents[kind] = value
		}
	}
	add("text", s.Text != nil, s.Text)
	add("image", s.Image != nil, s.Image)
	add("video", s.Video != nil, s.Video)
	add("audio", s.Audio != nil, s.Audio)
	add("document", s.Document != nil, s.Document)
	add("sticker", s.Sticker != nil, s.Sticker)
	add("template", s.Template != nil, s.Template)
	add("interactive", s.Interactive != nil, s.Interactive)
	add("location", s.Location != nil, s.Location)
	add("contacts", len(s.Contacts) > 0, s.Contacts)
	add("reaction", s.Reaction != nil, s.Reaction)
	if len(contents) == 0 {
		return et.Json{}, errors.New(MSG_MESSAGE_EMPTY)
	}
	if len(contents) > 1 {
		kinds := []string{}
		for kind := range contents {
			kinds = append(kinds, kind)
		}
		return et.Json{}, fmt.Errorf(MSG_MESSAGE_AMBIGUOUS, strings.Join(kinds, ", "))
	}

	result := et.Json{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                strings.TrimPrefix(strings.TrimSpace(s.To), "+"),
	}
	for kind, value := range contents {
		result["type"] = kind
		result[kind] = value
	}
	if s.ReplyTo != "" {
		result["context"] = et.Json{"message_id": s.ReplyTo}
	}
	if s.CallbackData != "" {
		result["biz_opaque_callback_data"] = s.CallbackData
	}
	return result, nil
}

/**
* Send: POST /{phone_number_id}/messages
* @param ctx context.Context, message Message
* @return SendResult, error
**/
func (s *Client) Send(ctx context.Context, message Message) (SendResult, error) {
	path, err := s.phonePath("/messages")
	if err != nil {
		return SendResult{}, err
	}
	body, err := message.body()
	if err != nil {
		return SendResult{}, err
	}

	output, err := s.Call(ctx, http.MethodPost, path, nil, body)
	if err != nil {
		return SendResult{}, err
	}

	result := SendResult{Raw: output}
	if messages := output.ArrayJson("messages"); len(messages) > 0 {
		result.MessageId = messages[0].Str("id")
		result.MessageStatus = messages[0].Str("message_status")
	}
	if contacts := output.ArrayJson("contacts"); len(contacts) > 0 {
		result.WaId = contacts[0].Str("wa_id")
		result.Input = contacts[0].Str("input")
	}
	return result, nil
}

/**
* MarkRead: marca como leído un mensaje recibido (y los anteriores); typing muestra «escribiendo…» hasta la respuesta (máx. 25 s)
* @param ctx context.Context, messageId string, typing bool
* @return error
**/
func (s *Client) MarkRead(ctx context.Context, messageId string, typing bool) error {
	path, err := s.phonePath("/messages")
	if err != nil {
		return err
	}

	body := et.Json{"messaging_product": "whatsapp", "status": "read", "message_id": messageId}
	if typing {
		body["typing_indicator"] = et.Json{"type": "text"}
	}
	_, err = s.Call(ctx, http.MethodPost, path, nil, body)
	return err
}
