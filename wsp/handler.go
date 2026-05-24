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
* sendRequest
* @param message *Message
* @return et.Json, error
**/
func (s *Whatsapp) sendRequest(message *Message) (et.Json, error) {
	url := fmt.Sprintf("%s/%s/messages", s.Path, s.PhoneNumberId)
	body := message.body()
	if s.isDebug {
		logs.Debug("body:", body.ToString())
	}

	if s.isTest {
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

/**
* SendTextMessage
* @param to, text string
* @return et.Json, error
**/
func (s *Whatsapp) SendTextMessage(to, text string) (et.Json, error) {
	message := &Message{
		To:   to,
		Text: text,
	}
	message.kind = MessageTypeText

	return s.sendRequest(message)
}

/**
* SendReplyTextMessage
* @param to, message_id, text string
* @return et.Json, error
**/
func (s *Whatsapp) SendReplyTextMessage(to, message_id, text string) (et.Json, error) {
	message := &Message{
		To:        to,
		Text:      text,
		MessageID: message_id,
	}
	message.kind = MessageTypeReplyText

	return s.sendRequest(message)
}

/**
* SendTextMessageWithPreviewURL
* @param to, url, text string
* @return et.Json, error
**/
func (s *Whatsapp) SendTextMessageWithPreviewURL(to, url string) (et.Json, error) {
	message := &Message{
		To:  to,
		Url: Url{Url: url},
	}
	message.kind = MessageTypeTextWithPreviewURL

	return s.sendRequest(message)
}

/**
* SendReplyWithReactionMessage
* @param to, message_id, emoji string
* @return et.Json, error
**/
func (s *Whatsapp) SendReplyWithReactionMessage(to, messageId, emoji string) (et.Json, error) {
	message := &Message{
		To:        to,
		MessageID: messageId,
		Emoji:     emoji,
	}
	message.kind = MessageTypeReplyWithReaction

	return s.sendRequest(message)
}
