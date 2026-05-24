package wsp

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/instances"
)

type Whatsapp struct {
	Path              string          `json:"path"`
	Token             string          `json:"token"`
	PhoneNumberId     string          `json:"phone_number_id"`
	VerifyToken       string          `json:"verify_token"`
	EventHandler      func(et.Json)   `json:"-"`
	EventHandlerError func(error)     `json:"-"`
	store             instances.Store `json:"-"`
	isDebug           bool            `json:"-"`
}

/**
* NewSender
* @param token string, phoneNumberId string
* @return *Whatsapp
**/
func NewSender(token, phoneNumberId string) *Whatsapp {
	return &Whatsapp{
		Path:          envar.GetStr("WHATSAPP_API_URL", "https://graph.facebook.com/v22.0"),
		Token:         token,
		PhoneNumberId: phoneNumberId,
		EventHandler:  nil,
		isDebug:       false,
	}
}

/**
* SetDebug
* @param isDebug bool
**/
func (w *Whatsapp) SetDebug(isDebug bool) {
	w.isDebug = isDebug
}

/**
* SetVerifyToken
* @param verifyToken string
**/
func (w *Whatsapp) SetVerifyToken(verifyToken string) {
	w.VerifyToken = verifyToken
}

/**
* SetEventHandler
* @param handler func(*Event)
**/
func (w *Whatsapp) SetEventHandler(fn func(et.Json)) {
	w.EventHandler = fn
}

/**
* SetEventHandlerError
* @param handler func(error)
**/
func (w *Whatsapp) SetEventHandlerError(fn func(error)) {
	w.EventHandlerError = fn
}
