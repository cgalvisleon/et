package ia

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type TypeConversation string

const (
	Direct TypeConversation = "direct"
	Group  TypeConversation = "group"
)

type Conversation struct {
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
	ID            string                  `json:"id"`
	ConvID        string                  `json:"conv_id"`
	Title         string                  `json:"title"`
	Type          TypeConversation        `json:"type"`
	LastMessage   *Message                `json:"last_message"`
	LimitMessages int                     `json:"limit_messages"`
	Messages      []*Message              `json:"-"`
	Participants  map[string]*Participant `json:"participants"`
	mu            sync.RWMutex            `json:"-"`
	to            *Participant            `json:"-"`
	ia            *Ia                     `json:"-"`
	isDebug       bool                    `json:"-"`
	isChanged     bool                    `json:"-"`
}

/**
* newConversation
* @param to *Participant, title string, conversationType TypeConversation
* @return *Conversation
**/
func newConversation(to *Participant, title string, conversationType TypeConversation) *Conversation {
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
		Title:         title,
		Type:          conversationType,
		LimitMessages: limitMessages,
		Messages:      make([]*Message, 0),
		Participants:  make(map[string]*Participant, 0),
		mu:            sync.RWMutex{},
		to:            to,
		ia:            to.ia,
		isDebug:       to.ia.isDebug,
	}
	return result
}

/**
* save
* @return error
**/
func (s *Conversation) save(userId string) error {
	s.UpdatedAt = timezone.Now()
	data := s.ToJson()
	data.Set("user_id", userId)
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.ia.store != nil {
		err := s.ia.store.Set(s.ID, "conversation", s.ia.TenantID, s.ia.ID, s, userId)
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
	if s.ia != nil && s.ia.store != nil {
		err := s.ia.store.Delete(s.ID, "conversation")
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

	items, err := s.ia.store.
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
* ToJson
* @return et.Json
**/
func (s *Conversation) ToJson() et.Json {
	return et.Json{
		"created_at":     timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":     timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"tenant_id":      s.ia.TenantID,
		"owner_id":       s.ia.ID,
		"id":             s.ID,
		"conv_id":        s.ConvID,
		"title":          s.Title,
		"type":           s.Type,
		"last_message":   s.LastMessage,
		"limit_messages": s.LimitMessages,
		"participants":   s.Participants,
		"messages":       s.Messages,
		"to": et.Json{
			"id":   s.to.ID,
			"to":   s.to.To,
			"name": s.to.Name,
		},
	}
}

/**
* AddParticipant
* @param participant *Participant
* @return *Conversation
**/
func (s *Conversation) AddParticipant(participant *Participant) *Conversation {
	_, ok := s.Participants[participant.To]
	if ok {
		return s
	}

	s.Participants[participant.To] = participant
	s.isChanged = true
	return s
}

/**
* SetConvId
* @param convId string
* @return *Conversation
**/
func (s *Conversation) SetConvId(convId string) *Conversation {
	if s.ConvID == convId {
		return s
	}
	s.ConvID = convId
	s.isChanged = true
	return s
}

/**
* SetLimitMessages
* @param limit int
* @return *Conversation
**/
func (s *Conversation) SetLimitMessages(limit int) *Conversation {
	if s.LimitMessages == limit {
		return s
	}
	s.LimitMessages = limit
	s.isChanged = true
	return s
}

/**
* SendTextMessage
* @param content string
* @return (*Message, error)
**/
func (s *Conversation) SendTextMessage(content, userId string) (*Message, error) {
	if s.ia.sender == nil {
		return nil, fmt.Errorf(MSG_SENDER_NOT_FOUND)
	}

	ms := newMessage(s, s.to.UserID, s.to.To, Text, content)
	ms.setStatus(Sent, userId)
	s.Messages = append(s.Messages, ms)
	s.LastMessage = ms
	_, err := s.ia.sender.SendTextMessage(ms.To, ms.Content)
	if err != nil {
		ms.setStatus(Failed, userId)
		return nil, err
	}

	err = ms.setStatus(Delivered, userId)
	if err != nil {
		return nil, err
	}

	return ms, nil
}
