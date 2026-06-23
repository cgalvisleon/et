package jia

import (
	"errors"
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
	DIRECT TypeConversation = "direct"
	GROUP  TypeConversation = "group"
)

type Conversation struct {
	CreatedAt     time.Time                                `json:"created_at"`
	UpdatedAt     time.Time                                `json:"updated_at"`
	IAID          string                                   `json:"ia_id"`
	ID            string                                   `json:"id"`
	ConvID        string                                   `json:"conv_id"`
	Title         string                                   `json:"title"`
	Type          TypeConversation                         `json:"type"`
	LastMessage   *Message                                 `json:"last_message"`
	LimitMessages int                                      `json:"limit_messages"`
	Messages      []*Message                               `json:"Messages"`
	To            *Participant                             `json:"to"`
	Participants  map[string]*Participant                  `json:"participants"`
	AuditLog      []et.Json                                `json:"audit_log"`
	mu            sync.RWMutex                             `json:"-"`
	ia            *Ia                                      `json:"-"`
	isDebug       bool                                     `json:"-"`
	isChanged     bool                                     `json:"-"`
	store         Store                                    `json:"-"`
	onSave        []func(conversation *Conversation) error `json:"-"`
	onDelete      []func(conversation *Conversation) error `json:"-"`
}

/**
* newConversation
* @param to *Participant, title string, conversationType TypeConversation
* @return *Conversation
**/
func (s *Ia) newConversation(to *Participant, title string, conversationType TypeConversation) *Conversation {
	if title == "" {
		title = to.Name
	}

	limitMessages := envar.GetInt("LIMIT_MESSAGES", 100)
	id := reg.UUID()
	now := timezone.Now()
	result := &Conversation{
		CreatedAt:     now,
		UpdatedAt:     now,
		IAID:          s.ID,
		ID:            id,
		ConvID:        to.ConvID,
		Title:         title,
		Type:          conversationType,
		LimitMessages: limitMessages,
		Messages:      make([]*Message, 0),
		To:            to,
		Participants:  make(map[string]*Participant, 0),
		AuditLog:      make([]et.Json, 0),
	}
	result.Participants[to.To] = to
	s.addConversation(result, to.ID)
	result.up(s)
	return result
}

/**
* loadConversation
* @param to string, dest *Conversation
* @return (bool, error)
**/
func (s *Ia) loadConversation(to *Participant) (*Conversation, error) {
	var result *Conversation
	exists, err := s.store.Get(to.To, "conversation", &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(MSG_CONVERSATION_NOT_FOUND)
	}

	result.up(s)
	return result, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Conversation) ToJson() et.Json {
	messages := []et.Json{}
	for _, message := range s.Messages {
		messages = append(messages, message.ToJson())
	}
	participants := []et.Json{}
	for _, participant := range s.Participants {
		participants = append(participants, participant.ToJson())
	}
	to := et.Json{
		"id":   s.To.ID,
		"to":   s.To.To,
		"name": s.To.Name,
	}
	return et.Json{
		"created_at":     timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":     timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"ia_id":          s.IAID,
		"id":             s.ID,
		"conv_id":        s.ConvID,
		"title":          s.Title,
		"type":           s.Type,
		"last_message":   s.LastMessage.ToJson(),
		"limit_messages": s.LimitMessages,
		"messages":       messages,
		"participants":   participants,
		"to":             to,
	}
}

/**
* ToString
* @return string
**/
func (s *Conversation) ToString() string {
	return s.ToJson().ToString()
}

/**
* up
* @param to *Participant
**/
func (s *Conversation) up(ia *Ia) error {
	s.mu = sync.RWMutex{}
	s.ia = ia
	s.isDebug = ia.isDebug
	s.store = ia.store
	s.onSave = make([]func(conversation *Conversation) error, 0)
	s.onDelete = make([]func(conversation *Conversation) error, 0)
	s.OnSave(func(conversation *Conversation) error {
		key := fmt.Sprintf("conversation:%s", conversation.ID)
		event.Publish(key, conversation.ToJson())
		return nil
	})
	s.OnDelete(func(conversation *Conversation) error {
		key := fmt.Sprintf("conversation:%s:delete", conversation.ID)
		event.Publish(key, et.Json{
			"id": conversation.ID,
		})
		return nil
	})
	for _, message := range s.Messages {
		s.Messages = append(s.Messages, message)
	}

	return nil
}

/**
* OnSave
* @param fn func(conversation *Conversation) error
* @return *Conversation
**/
func (s *Conversation) OnSave(fn func(conversation *Conversation) error) *Conversation {
	if s.onSave == nil {
		s.onSave = make([]func(conversation *Conversation) error, 0)
	}
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(conversation *Conversation) error
* @return *Conversation
**/
func (s *Conversation) OnDelete(fn func(conversation *Conversation) error) *Conversation {
	if s.onDelete == nil {
		s.onDelete = make([]func(conversation *Conversation) error, 0)
	}
	s.onDelete = append(s.onDelete, fn)
	return s
}

/**
* save
* @return error
**/
func (s *Conversation) save() error {
	if s.store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	s.isChanged = false

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	err := s.store.Set("conversation", s.ID, s.IAID, s)
	if err != nil {
		return err
	}

	for _, onSave := range s.onSave {
		err := onSave(s)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* delete
* @return error
**/
func (s *Conversation) delete() error {
	if s.store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	err := s.store.Delete("conversation", s.ID)
	if err != nil {
		return err
	}

	for _, onDelete := range s.onDelete {
		err := onDelete(s)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* AddParticipant
* @param participant *Participant
* @return *Conversation
**/
func (s *Conversation) AddParticipant(participant *Participant) *Conversation {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Participants[participant.To] = participant
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConvID = convId
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LimitMessages = limit
	return s
}

/**
* SendTextMessage
* @param content, userId, to string
* @return (*Message, error)
**/
func (s *Conversation) SendTextMessage(content, userId, to string) (*Message, error) {
	if s.ia.sender == nil {
		return nil, errors.New(MSG_SENDER_NOT_FOUND)
	}

	ms := s.ia.newMessage(s, userId, to, Text, content)
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
