package wsp

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

type Whatsapp struct {
	path              string        `json:"-"`
	token             string        `json:"-"`
	phone_number_id   string        `json:"-"`
	verifyToken       string        `json:"-"`
	eventHandler      func(et.Json) `json:"-"`
	eventHandlerError func(error)   `json:"-"`
	isDebug           bool          `json:"-"`
}

/**
* NewWhatsapp
* @param token string, phoneNumberId string
* @return *Whatsapp
**/
func NewWhatsapp(token, phoneNumberId string) *Whatsapp {
	return &Whatsapp{
		path:            envar.GetStr("WHATSAPP_API_URL", "https://graph.facebook.com/v22.0"),
		token:           token,
		phone_number_id: phoneNumberId,
		eventHandler:    nil,
		isDebug:         false,
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
	w.verifyToken = verifyToken
}

/**
* SetEventHandler
* @param handler func(*Event)
**/
func (w *Whatsapp) SetEventHandler(fn func(et.Json)) {
	w.eventHandler = fn
}

/**
* SetEventHandlerError
* @param handler func(error)
**/
func (w *Whatsapp) SetEventHandlerError(fn func(error)) {
	w.eventHandlerError = fn
}

/**
* webhooks
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Whatsapp) Webhooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mode := request.Query(r, "hub.mode").Str()
		token := request.Query(r, "hub.verify_token").Str()
		challenge := request.Query(r, "hub.challenge").Str()
		if mode == "subscribe" && token == s.verifyToken {
			if s.eventHandler != nil {
				s.eventHandler(et.Json{
					"kind": mode,
					"data": et.Json{
						"hub.mode":         mode,
						"hub.challenge":    challenge,
						"hub.verify_token": token,
					},
				})
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			return
		}
		response.HTTPError(w, r, http.StatusForbidden, http.StatusText(http.StatusForbidden))
	case http.MethodPost:
		body, err := request.GetBody(r)
		if err != nil {
			if s.eventHandlerError != nil {
				s.eventHandlerError(err)
			}
			return
		}
		if s.eventHandler != nil {
			s.eventHandler(et.Json{
				"kind": "conversation",
				"data": body,
			})
		}
		w.WriteHeader(http.StatusOK)
	default:
		response.HTTPError(w, r, http.StatusMethodNotAllowed, msg.MSG_METHOD_NOT_ALLOWED)
		return
	}
}

/**
* SendMessage
* @param to string, message *Message
* @return et.Json, error
**/
func (w *Whatsapp) SendMessage(to string, message *Message) (et.Json, error) {
	url := fmt.Sprintf("%s/%s/messages", w.path, w.phone_number_id)
	header := et.Json{
		"Authorization": "Bearer " + w.token,
		"Content-Type":  "application/json",
	}
	message.setTo(to)
	body := message.body()
	if w.isDebug {
		logs.Debug("body:", body.ToString())
	}

	res, status := request.Post(url, header, body)
	if status.Code != 200 {
		return et.Json{}, errors.New(status.Message)
	}

	result, err := res.ToJson()
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}
