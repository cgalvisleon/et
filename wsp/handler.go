package wsp

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

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
		if mode == "subscribe" && token == s.VerifyToken {
			if s.EventHandler != nil {
				s.EventHandler(et.Json{
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
			if s.EventHandlerError != nil {
				s.EventHandlerError(err)
			}
			return
		}
		if s.EventHandler != nil {
			s.EventHandler(et.Json{
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
* getHeader
* @return et.Json
**/
func (s *Whatsapp) getHeader() et.Json {
	header := et.Json{
		"Authorization": "Bearer " + s.Token,
		"Content-Type":  "application/json",
	}
	return header
}

/**
* SendMessage
* @param message *Message
* @return et.Json, error
**/
func (s *Whatsapp) SendMessage(message *Message) (et.Json, error) {
	url := fmt.Sprintf("%s/%s/messages", s.Path, s.PhoneNumberId)
	message.kind = MessageTypeText
	body := message.body()
	if s.isDebug {
		logs.Debug("body:", body.ToString())
	}

	notSend := body.Bool("not_send")
	if notSend {
		return body, nil
	}

	res, status := request.Post(url, s.getHeader(), body)
	if status.Code != 200 {
		return et.Json{}, errors.New(status.Message)
	}

	result, err := res.ToJson()
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}
