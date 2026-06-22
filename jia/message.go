package jia

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type StatusMessage string

const (
	Sent      StatusMessage = "sent"
	Delivered StatusMessage = "delivered"
	Read      StatusMessage = "read"
	Failed    StatusMessage = "failed"
)

type TypeMessage string

const (
	Text  TypeMessage = "text"
	Image TypeMessage = "image"
	Video TypeMessage = "video"
	Audio TypeMessage = "audio"
	File  TypeMessage = "file"
)

type MessageStatus struct {
	CreatedAt time.Time     `json:"read_at"`
	MessageID string        `json:"message_id"`
	UserID    string        `json:"user_id"`
	Status    StatusMessage `json:"status"`
}

type Message struct {
	CreatedAt       time.Time                      `json:"created_at"`
	IAID            string                         `json:"ia_id"`
	ConversationID  string                         `json:"conversation_id"`
	ID              string                         `json:"id"`
	Type            TypeMessage                    `json:"type"`
	UserID          string                         `json:"user_id"`
	To              string                         `json:"to"`
	Content         string                         `json:"content"`
	LastStatus      *MessageStatus                 `json:"last_status"`
	MessageStatuses []*MessageStatus               `json:"message_statuses"`
	conversation    *Conversation                  `json:"-"`
	isDebug         bool                           `json:"-"`
	ia              *Ia                            `json:"-"`
	store           Store                          `json:"-"`
	onSave          []func(message *Message) error `json:"-"`
	onDelete        []func(message *Message) error `json:"-"`
}

/**
* newMessage
* @param conversation *Conversation, userID, to string, tp TypeMessage, content string
* @return *Message
**/
func (s *Ia) newMessage(conversation *Conversation, userID, to string, tp TypeMessage, content string) *Message {
	now := timezone.Now()
	result := &Message{
		CreatedAt:       now,
		IAID:            s.ID,
		ConversationID:  conversation.ID,
		ID:              reg.ULID(),
		UserID:          userID,
		To:              to,
		Type:            tp,
		Content:         content,
		MessageStatuses: make([]*MessageStatus, 0),
		conversation:    conversation,
		isDebug:         s.isDebug,
		ia:              s,
		store:           s.store,
	}
	return result.up(s)
}

/**
* up
* @param ia *Ia
* @return *Message
**/
func (s *Message) up(ia *Ia) *Message {
	s.ia = ia
	s.isDebug = ia.isDebug
	s.store = ia.store
	s.onSave = make([]func(message *Message) error, 0)
	s.onDelete = make([]func(message *Message) error, 0)
	s.OnSave(func(message *Message) error {
		key := fmt.Sprintf("message:%s", message.ID)
		event.Publish(key, message.ToJson())
		return nil
	})
	s.OnDelete(func(message *Message) error {
		key := fmt.Sprintf("message:%s:delete", message.ID)
		event.Publish(key, et.Json{
			"id": message.ID,
		})
		return nil
	})
	return s
}

/**
* ToJson
* @return et.Json
**/
func (s *Message) ToJson() et.Json {
	return et.Json{
		"created_at":       timezone.Format(s.CreatedAt, timezone.RFC3339),
		"ia_id":            s.IAID,
		"conversation_id":  s.ConversationID,
		"id":               s.ID,
		"type":             s.Type,
		"user_id":          s.UserID,
		"to":               s.To,
		"content":          s.Content,
		"last_status":      s.LastStatus,
		"message_statuses": s.MessageStatuses,
	}
}

/**
* ToString
* @return string
**/
func (s *Message) ToString() string {
	return s.ToJson().ToString()
}

/**
* OnSave
* @param fn func(message *Message) error
* @return *Message
**/
func (s *Message) OnSave(fn func(message *Message) error) *Message {
	if s.onSave == nil {
		s.onSave = make([]func(message *Message) error, 0)
	}
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(message *Message) error
* @return *Message
**/
func (s *Message) OnDelete(fn func(message *Message) error) *Message {
	if s.onDelete == nil {
		s.onDelete = make([]func(message *Message) error, 0)
	}
	s.onDelete = append(s.onDelete, fn)
	return s
}

/**
* save
* @return error
**/
func (s *Message) save() error {
	if s.store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	err := s.store.Set("message", s.ID, s.IAID, s)
	if err != nil {
		return err
	}

	return nil
}

/**
* delete
* @return error
**/
func (s *Message) delete() error {
	if s.ia != nil && s.ia.store != nil {
		err := s.ia.store.Delete(s.ID, "message")
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_MESSAGE_DELETE, et.Json{
		"id": s.ID,
	})

	return nil
}

/**
* setStatus
* @param status StatusMessage
* @return error
**/
func (s *Message) setStatus(status StatusMessage) error {
	s.LastStatus = &MessageStatus{
		CreatedAt: timezone.Now(),
		MessageID: s.ID,
		UserID:    s.UserID,
		Status:    status,
	}
	s.MessageStatuses = append(s.MessageStatuses, s.LastStatus)
	return s.save()
}
