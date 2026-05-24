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
	isTest            bool            `json:"-"`
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
* Debug
* @return *Whatsapp
**/
func (s *Whatsapp) Debug() *Whatsapp {
	s.isDebug = true
	return s
}

/**
* IsTest
* @param isTest bool
* @return *Whatsapp
**/
func (s *Whatsapp) IsTest(isTest bool) *Whatsapp {
	s.isTest = isTest
	return s
}

/**
* SetVerifyToken
* @param verifyToken string
* @return *Whatsapp
**/
func (s *Whatsapp) SetVerifyToken(verifyToken string) *Whatsapp {
	s.VerifyToken = verifyToken
	return s
}

/**
* SetEventHandler
* @param handler func(*Event)
* @return *Whatsapp
**/
func (s *Whatsapp) SetEventHandler(fn func(et.Json)) *Whatsapp {
	s.EventHandler = fn
	return s
}

/**
* SetEventHandlerError
* @param handler func(error)
* @return *Whatsapp
**/
func (s *Whatsapp) SetEventHandlerError(fn func(error)) *Whatsapp {
	s.EventHandlerError = fn
	return s
}
