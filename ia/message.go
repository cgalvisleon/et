package ia

import (
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type TypeConversation string

const (
	Direct TypeConversation = "direct"
	Group  TypeConversation = "group"
)

type StatusMessage string

const (
	Sent      StatusMessage = "sent"
	Delivered StatusMessage = "delivered"
	Read      StatusMessage = "read"
)

type TypeMessage string

const (
	Text  TypeMessage = "text"
	Image TypeMessage = "image"
	Video TypeMessage = "video"
	Audio TypeMessage = "audio"
	File  TypeMessage = "file"
)

type Role string

const (
	Admin  Role = "admin"
	Member Role = "member"
)

type Participant struct {
	JoinedAt       time.Time `json:"joined_at"`
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	UserID         string    `json:"user_id"`
	To             string    `json:"to"`
	Role           Role      `json:"role"`
}

type MessageStatus struct {
	CreatedAt time.Time     `json:"read_at"`
	MessageID string        `json:"message_id"`
	UserID    string        `json:"user_id"`
	Status    StatusMessage `json:"status"`
}

type Message struct {
	CreatedAt       time.Time                 `json:"created_at"`
	ID              string                    `json:"id"`
	ConversationID  string                    `json:"conversation_id"`
	SenderID        string                    `json:"sender_id"`
	Type            TypeMessage               `json:"type"`
	Content         string                    `json:"content"`
	MessageStatuses map[string]*MessageStatus `json:"message_statuses"`
	ia              *Ia                       `json:"-"`
	isDebug         bool                      `json:"-"`
}

/**
* newMessage
* @param ia *Ia, conversationID, senderID string, tp TypeMessage, content string
* @return *Message
**/
func newMessage(ia *Ia, conversationID, senderID string, tp TypeMessage, content string) *Message {
	id := reg.GenUUId("message")
	return &Message{
		CreatedAt:       time.Now(),
		ID:              id,
		ConversationID:  conversationID,
		SenderID:        senderID,
		Type:            tp,
		Content:         content,
		MessageStatuses: map[string]*MessageStatus{},
		ia:              ia,
		isDebug:         ia.isDebug,
	}
}

/**
* ToJson
* @return et.Json
**/
func (s *Message) ToJson() et.Json {
	return et.Json{
		"created_at":       s.CreatedAt,
		"id":               s.ID,
		"conversation_id":  s.ConversationID,
		"sender_id":        s.SenderID,
		"type":             s.Type,
		"content":          s.Content,
		"message_statuses": s.MessageStatuses,
	}
}

/**
* save
* @return error
**/
func (s *Message) save() error {
	if s.isDebug {
		logs.Log(packageName, "save:", s.ToJson().ToString())
	}

	if s.ia != nil && s.ia.store != nil {
		return s.ia.store.Set(s.ID, "message", s)
	}

	return nil
}

/**
* setStatus
* @param userID string, status StatusMessage
* @return error
**/
func (s *Message) setStatus(userID string, status StatusMessage) error {
	s.MessageStatuses[userID] = &MessageStatus{
		CreatedAt: timezone.Now(),
		MessageID: s.ID,
		UserID:    userID,
		Status:    status,
	}
	return s.save()
}
