package jwsp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/cgalvisleon/et/et"
)

// Los tipos de evento del webhook
const (
	EventMessage = "message"
	EventStatus  = "status"
	EventError   = "error"
)

// SignatureHeader, signaturePrefix: el encabezado de la firma de Meta y su prefijo (sha256=<hex>)
const (
	SignatureHeader = "X-Hub-Signature-256"
	signaturePrefix = "sha256="
)

// MaxWebhookBody: el cuerpo más grande que ParseWebhook lee (5 MB)
const MaxWebhookBody = 5 << 20

// WebhookError: un error en el webhook (value.errors, messages[].errors o statuses[].errors); Details es error_data.details
type WebhookError struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Details string `json:"details"`
	Href    string `json:"href"`
}

// Reply: la respuesta a un mensaje con botones: un botón de plantilla (Type button: Id es el payload) o una respuesta
// interactiva (button_reply o list_reply)
type Reply struct {
	Type        string `json:"type"`
	Id          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// IncomingMessage: un mensaje que un usuario le escribió al número. Text es el texto, el del botón o el título de la
// respuesta; Content es el objeto de su tipo tal como llegó (image, location…); From puede venir vacío si el usuario solo
// tiene usuario (BSUID: FromUserId)
type IncomingMessage struct {
	MessageId  string         `json:"message_id"`
	From       string         `json:"from"`
	FromUserId string         `json:"from_user_id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Text       string         `json:"text"`
	Reply      *Reply         `json:"reply,omitempty"`
	Context    et.Json        `json:"context,omitempty"`
	Content    et.Json        `json:"content,omitempty"`
	Timestamp  int64          `json:"timestamp"`
	Errors     []WebhookError `json:"errors,omitempty"`
}

// Status: el estado de un mensaje enviado (sent, delivered, read, played o failed). CallbackData es el
// biz_opaque_callback_data con que se envió; Conversation y Pricing van tal como llegan
type Status struct {
	MessageId       string         `json:"message_id"`
	Status          string         `json:"status"`
	Recipient       string         `json:"recipient"`
	RecipientUserId string         `json:"recipient_user_id"`
	Timestamp       int64          `json:"timestamp"`
	CallbackData    string         `json:"callback_data,omitempty"`
	Conversation    et.Json        `json:"conversation,omitempty"`
	Pricing         et.Json        `json:"pricing,omitempty"`
	Errors          []WebhookError `json:"errors,omitempty"`
}

// Event: un evento del webhook con el número (phone_number_id) y la cuenta (waba_id) a los que llegó; solo uno de
// Message, Status o Error según Kind
type Event struct {
	Kind          string
	PhoneNumberId string
	WabaId        string
	DisplayPhone  string
	Message       *IncomingMessage
	Status        *Status
	Error         *WebhookError
}

// errorItemWire: un error como lo manda Meta en el webhook
type errorItemWire struct {
	Code      int    `json:"code"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	ErrorData struct {
		Details string `json:"details"`
	} `json:"error_data"`
	Href string `json:"href"`
}

// webhookWire: el cuerpo del webhook como lo manda Meta (solo lo que se lee)
type webhookWire struct {
	Object string `json:"object"`
	Entry  []struct {
		Id      string `json:"id"`
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				Metadata struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberId      string `json:"phone_number_id"`
				} `json:"metadata"`
				Contacts []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaId   string `json:"wa_id"`
					UserId string `json:"user_id"`
				} `json:"contacts"`
				Messages []map[string]json.RawMessage `json:"messages"`
				Statuses []struct {
					Id                    string          `json:"id"`
					Status                string          `json:"status"`
					Timestamp             string          `json:"timestamp"`
					RecipientId           string          `json:"recipient_id"`
					RecipientUserId       string          `json:"recipient_user_id"`
					BizOpaqueCallbackData string          `json:"biz_opaque_callback_data"`
					Conversation          et.Json         `json:"conversation"`
					Pricing               et.Json         `json:"pricing"`
					Errors                []errorItemWire `json:"errors"`
				} `json:"statuses"`
				Errors []errorItemWire `json:"errors"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

/**
* VerifyTokenOk: si el hub.verify_token de la suscripción es el esperado (tiempo constante; vacío nunca vale)
* @param got, want string
* @return bool
**/
func VerifyTokenOk(got, want string) bool {
	if got == "" || want == "" {
		return false
	}
	return hmac.Equal([]byte(got), []byte(want))
}

/**
* VerifyHandler: el GET de suscripción: con hub.mode subscribe y el verify token responde 200 con hub.challenge en texto; si no, 403
* @param verifyToken string
* @return http.HandlerFunc
**/
func VerifyHandler(verifyToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		query := r.URL.Query()
		if query.Get("hub.mode") != "subscribe" || !VerifyTokenOk(query.Get("hub.verify_token"), verifyToken) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, query.Get("hub.challenge"))
	}
}

/**
* Signature: la firma de un cuerpo como la manda Meta (sha256=<hex>); para las pruebas
* @param raw []byte, appSecret string
* @return string
**/
func Signature(raw []byte, appSecret string) string {
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(raw)
	return signaturePrefix + hex.EncodeToString(mac.Sum(nil))
}

/**
* VerifySignature: si signature (X-Hub-Signature-256) es el HMAC-SHA256 del cuerpo crudo con el app secret, en tiempo constante. Sin firma o sin app
* secret, false
* @param raw []byte, signature, appSecret string
* @return bool
**/
func VerifySignature(raw []byte, signature, appSecret string) bool {
	if appSecret == "" || !strings.HasPrefix(signature, signaturePrefix) {
		return false
	}

	got, err := hex.DecodeString(strings.TrimPrefix(signature, signaturePrefix))
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(raw)
	return hmac.Equal(got, mac.Sum(nil))
}

/**
* ReadWebhook: el cuerpo crudo de un POST del webhook (hasta MaxWebhookBody; más, ErrBodyTooLarge)
* @param r *http.Request
* @return []byte, error
**/
func ReadWebhook(r *http.Request) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, MaxWebhookBody+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxWebhookBody {
		return nil, ErrBodyTooLarge
	}
	return raw, nil
}

/**
* ParseWebhook: lee el POST, verifica X-Hub-Signature-256 con el app secret (ErrSignature si no vale) y devuelve sus eventos
* @param r *http.Request, appSecret string
* @return []Event, error
**/
func ParseWebhook(r *http.Request, appSecret string) ([]Event, error) {
	raw, err := ReadWebhook(r)
	if err != nil {
		return nil, err
	}
	if !VerifySignature(raw, r.Header.Get(SignatureHeader), appSecret) {
		return nil, ErrSignature
	}
	return ParseEvents(raw)
}

/**
* Handler: el webhook completo: GET con VerifyHandler y POST con ParseWebhook (401 firma, 413 grande, 400 cuerpo inválido; 200 y
* después fn con los eventos, en la misma goroutine)
* @param verifyToken, appSecret string, fn func([]Event)
* @return http.HandlerFunc
**/
func Handler(verifyToken, appSecret string, fn func([]Event)) http.HandlerFunc {
	verify := VerifyHandler(verifyToken)
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			verify(w, r)
		case http.MethodPost:
			events, err := ParseWebhook(r, appSecret)
			switch {
			case errors.Is(err, ErrSignature):
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			case errors.Is(err, ErrBodyTooLarge):
				http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
				return
			case err != nil:
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			if fn != nil {
				fn(events)
			}
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

/**
* ParseEvents: los eventos de un cuerpo del webhook ya verificado (object whatsapp_business_account, campo messages): un mensaje por cada
* messages[], un estado por cada statuses[] y un error por cada value.errors[]; otros campos se ignoran
* @param raw []byte
* @return []Event, error
**/
func ParseEvents(raw []byte) ([]Event, error) {
	var body webhookWire
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}

	if body.Object != "whatsapp_business_account" {
		return nil, fmt.Errorf(MSG_WEBHOOK_OBJECT, body.Object)
	}

	result := []Event{}
	for _, entry := range body.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}

			value := change.Value
			base := Event{
				PhoneNumberId: value.Metadata.PhoneNumberId,
				WabaId:        entry.Id,
				DisplayPhone:  value.Metadata.DisplayPhoneNumber,
			}

			names := map[string]string{}
			for _, contact := range value.Contacts {
				names[contact.WaId] = contact.Profile.Name
				names[contact.UserId] = contact.Profile.Name
			}

			for _, item := range value.Messages {
				message := incomingOf(item)
				message.Name = nameOf(names, message.From, message.FromUserId)
				event := base
				event.Kind = EventMessage
				event.Message = &message
				result = append(result, event)
			}

			for _, item := range value.Statuses {
				status := Status{
					MessageId:       item.Id,
					Status:          item.Status,
					Recipient:       item.RecipientId,
					RecipientUserId: item.RecipientUserId,
					Timestamp:       unixOf(item.Timestamp),
					CallbackData:    item.BizOpaqueCallbackData,
					Conversation:    item.Conversation,
					Pricing:         item.Pricing,
					Errors:          webhookErrorsOf(item.Errors),
				}
				event := base
				event.Kind = EventStatus
				event.Status = &status
				result = append(result, event)
			}

			for _, failure := range webhookErrorsOf(value.Errors) {
				event := base
				event.Kind = EventError
				event.Error = &failure
				result = append(result, event)
			}
		}
	}

	return result, nil
}

/**
* incomingOf: un mensaje entrante desde su objeto: el texto según su tipo (text.body, button.text o el título de button_reply/list_reply), la
* respuesta, el contexto, el objeto de su tipo (Content) y sus errores (type unsupported)
* @param item map[string]json.RawMessage
* @return IncomingMessage
**/
func incomingOf(item map[string]json.RawMessage) IncomingMessage {
	text := func(key string) string {
		var value string
		json.Unmarshal(item[key], &value)
		return value
	}
	object := func(key string) et.Json {
		value := et.Json{}
		json.Unmarshal(item[key], &value)
		return value
	}

	tp := text("type")
	result := IncomingMessage{
		MessageId:  text("id"),
		From:       text("from"),
		FromUserId: text("from_user_id"),
		Type:       tp,
		Timestamp:  unixOf(text("timestamp")),
	}
	if context := object("context"); len(context) > 0 {
		result.Context = context
	}

	content := object(tp)
	switch tp {
	case "text":
		result.Text = content.Str("body")
	case "button":
		result.Text = content.Str("text")
		result.Reply = &Reply{Type: "button", Id: content.Str("payload"), Title: content.Str("text")}
	case "interactive":
		kind := content.Str("type")
		reply := content.Json(kind)
		if kind == "button_reply" || kind == "list_reply" {
			result.Text = reply.Str("title")
			result.Reply = &Reply{Type: kind, Id: reply.Str("id"), Title: reply.Str("title"), Description: reply.Str("description")}
		}
	}
	if len(content) > 0 {
		result.Content = content
	}

	var failures []errorItemWire
	json.Unmarshal(item["errors"], &failures)
	result.Errors = webhookErrorsOf(failures)

	return result
}

/**
* webhookErrorsOf: los errores del webhook con su detalle (error_data.details) en plano
* @param items []errorItemWire
* @return []WebhookError
**/
func webhookErrorsOf(items []errorItemWire) []WebhookError {
	result := []WebhookError{}
	for _, item := range items {
		failure := WebhookError{Code: item.Code, Title: item.Title, Message: item.Message, Details: item.ErrorData.Details, Href: item.Href}
		result = append(result, failure)
	}
	return result
}

/**
* nameOf: el nombre del perfil del contacto por su número o su BSUID
* @param names map[string]string, keys ...string
* @return string
**/
func nameOf(names map[string]string, keys ...string) string {
	for _, key := range keys {
		if name := names[key]; key != "" && name != "" {
			return name
		}
	}
	return ""
}

/**
* unixOf: un timestamp unix en texto (Meta los manda así); 0 si no es un número
* @param value string
* @return int64
**/
func unixOf(value string) int64 {
	result, _ := strconv.ParseInt(value, 10, 64)
	return result
}
