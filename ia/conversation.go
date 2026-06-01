package ia

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type TypeConversation string

const (
	Direct TypeConversation = "direct"
	Group  TypeConversation = "group"
)

type Conversation struct {
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	ID            string           `json:"id"`
	ConvID        string           `json:"conv_id"`
	OwnerID       string           `json:"owner_id"`
	Title         string           `json:"title"`
	Type          TypeConversation `json:"type"`
	LastMessage   *Message         `json:"last_message"`
	LimitMessages int              `json:"limit_messages"`
	Messages      []*Message       `json:"-"`
	messageStore  jsql.Store       `json:"-"`
	mu            sync.RWMutex     `json:"-"`
	to            *Participant     `json:"-"`
	ia            *Ia              `json:"-"`
	isDebug       bool             `json:"-"`
}

/**
* newConversation
* @param ia *Ia, ownerId string, title string, type TypeConversation
* @return (*Conversation, error)
**/
func newConversation(to *Participant, title string, conversationType TypeConversation) (*Conversation, error) {
	if !utility.ValidStr(title, 4, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "title")
	}

	if title == "" {
		title = to.Name
	}

	limitMessages := envar.GetInt("LIMIT_MESSAGES", 100)
	id := reg.GenULID("conversation")
	now := timezone.Now()
	result := &Conversation{
		CreatedAt:     now,
		UpdatedAt:     now,
		ID:            id,
		ConvID:        id,
		OwnerID:       to.ID,
		Title:         title,
		Type:          conversationType,
		Messages:      make([]*Message, 0),
		LimitMessages: limitMessages,
		mu:            sync.RWMutex{},
		to:            to,
		ia:            to.ia,
		isDebug:       to.ia.isDebug,
	}
	var err error
	result.messageStore, err = jsql.DefineInstance(to.ia.db, "ia", "message", jsql.KindJson)
	if err != nil {
		return nil, err
	}

	err = result.save()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Conversation) ToJson() et.Json {
	return et.Json{
		"created_at":     s.CreatedAt,
		"updated_at":     s.UpdatedAt,
		"id":             s.ID,
		"conv_id":        s.ConvID,
		"owner_id":       s.OwnerID,
		"title":          s.Title,
		"type":           s.Type,
		"last_message":   s.LastMessage,
		"limit_messages": s.LimitMessages,
		"to": et.Json{
			"id":   s.to.ID,
			"to":   s.to.To,
			"name": s.to.Name,
		},
	}
}

/**
* save
* @return error
**/
func (s *Conversation) save() error {
	data := s.ToJson()
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.ia.conversationStore != nil {
		err := s.ia.conversationStore.Set(s.ID, "conversations", s.OwnerID, s)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_CONVERSATION_SET, data)

	return nil
}

/**
* delete
* @return error
**/
func (s *Conversation) delete() error {
	if s.ia != nil && s.ia.conversationStore != nil {
		err := s.ia.conversationStore.Delete(s.ID)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_CONVERSATION_DELETE, et.Json{
		"id": s.ID,
	})

	return nil
}

/**
* up
* @param to *Participant
**/
func (s *Conversation) up(to *Participant) error {
	s.to = to
	s.ia = to.ia
	s.isDebug = to.ia.isDebug

	items, err := s.messageStore.
		Query(et.Json{})
	if err != nil {
		return err
	}

	for _, item := range items.Result {
		var message *Message
		bt := []byte(item.ToString())
		err = json.Unmarshal(bt, message)
		if err != nil {
			return err
		}
		s.Messages = append(s.Messages, message)
	}

	return nil
}

/**
* SetConvId
* @param convId string
* @return error
**/
func (s *Conversation) SetConvId(convId string) error {
	if s.ConvID == convId {
		return nil
	}
	s.ConvID = convId
	return s.save()
}

/**
* SetLimitMessages
* @param limit int
* @return error
**/
func (s *Conversation) SetLimitMessages(limit int) error {
	if s.LimitMessages == limit {
		return nil
	}
	s.LimitMessages = limit
	return s.save()
}

/**
* SendTextMessage
* @param content string
* @return (*Message, error)
**/
func (s *Conversation) SendTextMessage(content string) (*Message, error) {
	if s.ia.sender == nil {
		return nil, fmt.Errorf(MSG_SENDER_NOT_FOUND)
	}

	ms := newMessage(s, s.to.UserID, s.to.To, Text, content)
	ms.setStatus(Sent)
	s.Messages = append(s.Messages, ms)
	s.LastMessage = ms
	_, err := s.ia.sender.SendTextMessage(ms.To, ms.Content)
	if err != nil {
		ms.setStatus(Failed)
		return nil, err
	}

	err = ms.setStatus(Delivered)
	if err != nil {
		return nil, err
	}

	return ms, nil
}
