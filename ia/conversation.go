package ia

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/instances"
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
}

func newMessage(conversationID, senderID string, tp TypeMessage, content string) *Message {
	return &Message{
		CreatedAt:       time.Now(),
		ID:              utility.UUID(),
		ConversationID:  conversationID,
		SenderID:        senderID,
		Type:            tp,
		Content:         content,
		MessageStatuses: map[string]*MessageStatus{},
	}
}

/**
* setStatus
* @param userID string, status StatusMessage
**/
func (s *Message) setStatus(userID string, status StatusMessage) {
	s.MessageStatuses[userID] = &MessageStatus{
		CreatedAt: timezone.Now(),
		MessageID: s.ID,
		UserID:    userID,
		Status:    status,
	}
}

/**
* ToJson
* @return (et.Json, error)
**/
func (s *Message) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type Conversation struct {
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
	ID           string                  `json:"id"`
	Type         TypeConversation        `json:"type"`
	Participants map[string]*Participant `json:"participants"`
	Messages     []*Message              `json:"messages"`
	LastMessage  *Message                `json:"last_message"`
	owner        *Conversations          `json:"-"`
}

/**
* save
* @return error
**/
func (s *Conversation) save() error {
	return s.owner.setInstance(s.ID, "conversations", s)
}

/**
* setMessage
* @param to string, tp TypeMessage, content string
* @return error
**/
func (s *Conversation) setMessage(to string, tp TypeMessage, content string) (et.Item, error) {
	userId := fmt.Sprintf("%s:%s", s.owner.participantPrefix, to)
	_, exists := s.Participants[userId]
	if !exists {
		_, err := s.owner.getParticipant(to)
		if err != nil {
			return et.Item{}, err
		}

		now := timezone.Now()
		participant := &Participant{
			JoinedAt:       now,
			ID:             reg.GenULID("participant"),
			ConversationID: s.ID,
			UserID:         userId,
			To:             to,
			Role:           Member,
		}

		err = s.owner.setInstance(userId, "participants", participant)
		if err != nil {
			return et.Item{}, err
		}

		s.Participants[userId] = participant
	}

	ms := newMessage(s.ID, userId, tp, content)
	ms.setStatus(userId, Sent)
	s.Messages = append(s.Messages, ms)
	s.LastMessage = ms
	err := s.owner.setInstance(ms.ID, "messages", ms)
	if err != nil {
		return et.Item{}, err
	}

	result, err := ms.ToJson()
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok:     true,
		Result: result,
	}, nil
}

type Conversations struct {
	participantPrefix string                    `json:"-"`
	getInstance       instances.GetInstanceFn   `json:"-"`
	setInstance       instances.SetInstanceFn   `json:"-"`
	queryInstance     instances.QueryInstanceFn `json:"-"`
}

/**
* NewConversations
* @param participantPrefix string, store instances.Store
* @return (*Conversations, error)
**/
func NewConversations(participantPrefix string, store instances.Store) (*Conversations, error) {
	if !utility.ValidStr(participantPrefix, 4, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "participant_prefix")
	}

	if store == nil {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "store")
	}

	result := &Conversations{
		participantPrefix: participantPrefix,
	}

	result.getInstance = store.Get
	result.setInstance = store.Set
	result.queryInstance = store.Query

	return result, nil
}

/**
* getConversation
* @param id string, tp TypeConversation
* @return (*Conversation, error)
**/
func (s *Conversations) getConversation(id string, tp TypeConversation) (*Conversation, error) {
	if id == "" {
		id = reg.GenULID("conversation")
	}

	var result *Conversation
	exists, err := s.getInstance(id, &result)
	if err != nil {
		return nil, err
	}

	if exists {
		result.owner = s
		return result, nil
	}

	now := timezone.Now()
	resut := &Conversation{
		CreatedAt:    now,
		UpdatedAt:    now,
		ID:           id,
		Type:         tp,
		Participants: map[string]*Participant{},
		Messages:     []*Message{},
		LastMessage:  &Message{},
		owner:        s,
	}

	err = s.setInstance(id, "conversations", resut)
	if err != nil {
		return nil, err
	}

	return resut, nil
}

/**
* getParticipant
* @param phone string
* @return (et.Json, error)
**/
func (s *Conversations) getParticipant(phone string) (et.Json, error) {
	var result et.Json
	id := fmt.Sprintf("%s:%s", s.participantPrefix, phone)
	exists, err := s.getInstance(id, &result)
	if err != nil {
		return result, err
	}

	if exists {
		return result, nil
	}

	now := timezone.Now()
	result = et.Json{
		"created_at": now,
		"updated_at": now,
		"id":         id,
		"phone":      phone,
	}

	err = s.setInstance(id, s.participantPrefix, result)
	if err != nil {
		return result, err
	}

	return result, nil
}

/**
* SetMessage
* @param convID string, to string, tpContent TypeMessage, content string
* @return et.Item, error
**/
func (s *Conversations) SetMessage(convID, to string, tpContent TypeMessage, content string) (et.Item, error) {
	result, err := s.getConversation(convID, Direct)
	if err != nil {
		return et.Item{}, err
	}

	return result.setMessage(to, tpContent, content)
}

/**
* StatusMessage
* @param messageId string, userId string, status StatusMessage
* @return error
**/
func (s *Conversations) StatusMessage(messageId string, userId string, status StatusMessage) error {
	var ms *Message
	exists, err := s.getInstance(messageId, &ms)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("message not found")
	}

	ms.setStatus(userId, status)
	return s.setInstance(messageId, "messages", ms)
}
