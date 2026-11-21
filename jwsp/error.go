package jwsp

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// Códigos de error de la Graph API que el paquete reconoce
const (
	CodeReengagement     = 131047 // fuera de la ventana de 24 h: solo plantillas
	CodeRateLimit        = 130429 // límite de mensajes por segundo del número
	CodePairRateLimit    = 131056 // demasiados mensajes al mismo destinatario
	CodeSpamRateLimit    = 131048 // límite por calidad (spam)
	CodeTooManyCalls     = 4      // límite de llamadas de la app
	CodeUserTooManyCalls = 17     // límite de llamadas del usuario
	CodeBusinessThrottle = 80007  // límite de la cuenta de WhatsApp Business
	CodeUnknown          = 131000 // algo salió mal en Meta (transitorio)
	CodeServiceDown      = 131016 // servicio no disponible (transitorio)
	CodeRecipientInvalid = 131026 // el destinatario no puede recibir el mensaje
	CodeNotAllowedList   = 131030 // número fuera de la lista de prueba
	CodeTemplateMissing  = 132001 // la plantilla no existe en ese idioma
	CodeTemplateParams   = 132000 // el número de parámetros no coincide
)

// rateLimitCodes: los códigos que son límites de velocidad
var rateLimitCodes = []int{CodeRateLimit, CodePairRateLimit, CodeSpamRateLimit, CodeTooManyCalls, CodeUserTooManyCalls, CodeBusinessThrottle, 613}

// Error: un error de la Graph API ({error: {message, type, code, error_subcode, error_user_title, error_user_msg, error_data, fbtrace_id}})
// con el estado HTTP; Body es la respuesta cruda cuando Meta no mandó un error con esa forma
type Error struct {
	Status    int    `json:"status"`
	Code      int    `json:"code"`
	Subcode   int    `json:"error_subcode,omitempty"`
	Type      string `json:"type,omitempty"`
	Message   string `json:"message"`
	UserTitle string `json:"error_user_title,omitempty"`
	UserMsg   string `json:"error_user_msg,omitempty"`
	Details   string `json:"details,omitempty"`
	FbTraceId string `json:"fbtrace_id,omitempty"`
	Body      string `json:"body,omitempty"`
}

// errorWire: el error como lo manda Meta
type errorWire struct {
	Error *struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		Subcode   int    `json:"error_subcode"`
		UserTitle string `json:"error_user_title"`
		UserMsg   string `json:"error_user_msg"`
		FbTraceId string `json:"fbtrace_id"`
		ErrorData struct {
			Details string `json:"details"`
		} `json:"error_data"`
	} `json:"error"`
}

/**
* errorOf
* @param status int, raw []byte
* @return *Error
**/
func errorOf(status int, raw []byte) *Error {
	var wire errorWire
	if err := json.Unmarshal(raw, &wire); err != nil || wire.Error == nil {
		body := strings.TrimSpace(string(raw))
		return &Error{Status: status, Message: cmp.Or(body, http.StatusText(status)), Body: body}
	}

	item := wire.Error
	return &Error{
		Status:    status,
		Code:      item.Code,
		Subcode:   item.Subcode,
		Type:      item.Type,
		Message:   item.Message,
		UserTitle: item.UserTitle,
		UserMsg:   item.UserMsg,
		Details:   item.ErrorData.Details,
		FbTraceId: item.FbTraceId,
	}
}

/**
* Error
* @return string
**/
func (s *Error) Error() string {
	if s.Code == 0 {
		return fmt.Sprintf(MSG_META_STATUS, s.Status, s.Message)
	}
	return fmt.Sprintf(MSG_META_ERROR, s.Status, s.Text(), s.Code)
}

/**
* Text: el mensaje para el usuario si Meta lo manda; si no, el técnico
* @return string
**/
func (s *Error) Text() string {
	return cmp.Or(s.UserMsg, s.Message)
}

/**
* IsReengagement: fuera de la ventana de 24 h (131047): solo sale una plantilla
* @return bool
**/
func (s *Error) IsReengagement() bool {
	return s.Code == CodeReengagement
}

/**
* IsRateLimit: un límite de velocidad (HTTP 429 o uno de los códigos de límite)
* @return bool
**/
func (s *Error) IsRateLimit() bool {
	return s.Status == http.StatusTooManyRequests || slices.Contains(rateLimitCodes, s.Code)
}

/**
* IsTemporary: vale reintentar (límite de velocidad, 5xx o un código transitorio)
* @return bool
**/
func (s *Error) IsTemporary() bool {
	return s.IsRateLimit() || s.Status >= http.StatusInternalServerError || s.Code == CodeUnknown || s.Code == CodeServiceDown
}

/**
* AsError: el *Error de la Graph API dentro de err
* @param err error
* @return *Error, bool
**/
func AsError(err error) (*Error, bool) {
	var result *Error
	ok := errors.As(err, &result)
	return result, ok
}
