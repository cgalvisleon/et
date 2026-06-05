package ia

import (
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
	CreatedAt       time.Time        `json:"created_at"`
	ID              string           `json:"id"`
	ConversationID  string           `json:"conversation_id"`
	Type            TypeMessage      `json:"type"`
	UserID          string           `json:"user_id"`
	To              string           `json:"to"`
	Content         string           `json:"content"`
	LastStatus      *MessageStatus   `json:"last_status"`
	MessageStatuses []*MessageStatus `json:"message_statuses"`
	conversation    *Conversation    `json:"-"`
	isDebug         bool             `json:"-"`
	ia              *Ia              `json:"-"`
}

/**
* newMessage
* @param ia *Ia, conversationID, userID, to string, tp TypeMessage, content string
* @return *Message
**/
func newMessage(conversation *Conversation, userID, to string, tp TypeMessage, content string) *Message {
	id := reg.GenUUId("message")
	result := &Message{
		CreatedAt:       time.Now(),
		ID:              id,
		ConversationID:  conversation.ID,
		UserID:          userID,
		To:              to,
		Type:            tp,
		Content:         content,
		MessageStatuses: make([]*MessageStatus, 0),
		conversation:    conversation,
		isDebug:         conversation.isDebug,
		ia:              conversation.ia,
	}
	return result
}

/**
* save
* @return error
**/
func (s *Message) save(userId string) error {
	data := s.ToJson()
	data.Set("user_id", userId)
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	event.Publish(EVENT_MESSAGE_SET, data)

	if s.ia.store != nil {
		err := s.ia.store.Set(s.ID, "message", s.ia.TenantID, s.ia.ID, s, userId)
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
* ToJson
* @return et.Json
**/
func (s *Message) ToJson() et.Json {
	return et.Json{
		"created_at":       timezone.Format(s.CreatedAt, timezone.RFC3339),
		"tenant_id":        s.ia.TenantID,
		"owner_id":         s.ia.ID,
		"id":               s.ID,
		"conversation_id":  s.ConversationID,
		"user_id":          s.UserID,
		"to":               s.To,
		"type":             s.Type,
		"content":          s.Content,
		"message_statuses": s.MessageStatuses,
	}
}

/**
* setStatus
* @param status StatusMessage, userId string
* @return error
**/
func (s *Message) setStatus(status StatusMessage, userId string) error {
	s.LastStatus = &MessageStatus{
		CreatedAt: timezone.Now(),
		MessageID: s.ID,
		UserID:    s.UserID,
		Status:    status,
	}
	s.MessageStatuses = append(s.MessageStatuses, s.LastStatus)
	return s.save(userId)
}
