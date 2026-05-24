package ia

import (
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
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
	Name           string    `json:"name"`
	Role           Role      `json:"role"`
	ia             *Ia       `json:"-"`
}

/**
* newParticipant
* @param ia *Ia, conversationId, userId, to string, role Role
* @return *Participant
**/
func newParticipant(ia *Ia, conversationId, userId, to, name string, role Role) *Participant {
	return &Participant{
		JoinedAt:       timezone.Now(),
		ID:             reg.GenULID("participant"),
		ConversationID: conversationId,
		UserID:         userId,
		To:             to,
		Name:           name,
		Role:           role,
		ia:             ia,
	}
}

/**
* ToJson
* @return et.Json
**/
func (s *Participant) ToJson() et.Json {
	return et.Json{
		"joined_at":       s.JoinedAt,
		"id":              s.ID,
		"conversation_id": s.ConversationID,
		"user_id":         s.UserID,
		"to":              s.To,
		"role":            s.Role,
	}
}

/**
* save
* @return error
**/
func (s *Participant) save() error {
	data := s.ToJson()
	if s.ia.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.ia.store != nil {
		err := s.ia.store.Set(s.ID, "participant", s)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_PARTICIPANT_SET, data)

	return nil
}

/**
* delete
* @return error
**/
func (s *Participant) delete() error {
	if s.ia.store != nil {
		err := s.ia.store.Delete(s.ID)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_PARTICIPANT_DELETE, et.Json{
		"id": s.ID,
	})

	return nil
}

/**
* up
* @param ia *Ia
**/
func (s *Participant) up(ia *Ia) {
	s.ia = ia
}

type Conversation struct {
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
	ID            string                  `json:"id"`
	Title         string                  `json:"title"`
	Type          TypeConversation        `json:"type"`
	Participants  map[string]*Participant `json:"participants"`
	LastMessage   *Message                `json:"last_message"`
	Messages      []*Message              `json:"-"`
	LimitMessages int                     `json:"limit_messages"`
	mu            sync.RWMutex            `json:"-"`
	ia            *Ia                     `json:"-"`
	isDebug       bool                    `json:"-"`
}

/**
* newConversation
* @param ia *Ia, userId, to string, name string, title string, type TypeConversation
* @return (*Conversation, error)
**/
func newConversation(ia *Ia, userId, to, name, title string, conversationType TypeConversation) (*Conversation, error) {
	if !utility.ValidStr(to, 4, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "to")
	}
	if !utility.ValidStr(name, 4, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}
	if title == "" {
		title = name
	}

	limitMessages := envar.GetInt("LIMIT_MESSAGES", 100)
	id := reg.GenULID("conversation")
	now := timezone.Now()
	result := &Conversation{
		CreatedAt:     now,
		UpdatedAt:     now,
		ID:            id,
		Title:         title,
		Type:          conversationType,
		Participants:  make(map[string]*Participant),
		Messages:      make([]*Message, 0),
		LimitMessages: limitMessages,
		mu:            sync.RWMutex{},
		ia:            ia,
		isDebug:       ia.isDebug,
	}
	result.addParticipant(userId, to, name, Admin)

	return result, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Conversation) ToJson() et.Json {
	return et.Json{
		"id":           s.ID,
		"type":         s.Type,
		"participants": s.Participants,
		"last_message": s.LastMessage,
		"created_at":   s.CreatedAt,
		"updated_at":   s.UpdatedAt,
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

	if s.ia.store != nil {
		err := s.ia.store.Set(s.ID, "conversations", s)
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
		err := s.ia.store.Delete(s.ID)
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
* @param ia *Ia
**/
func (s *Conversation) up(ia *Ia) {
	s.ia = ia
	s.isDebug = ia.isDebug
}

/**
* setLimitMessages
* @param limit int
* @return error
**/
func (s *Conversation) setLimitMessages(limit int) error {
	s.LimitMessages = limit
	return s.save()
}

/**
* addParticipant
* @param userId, to, name string, role Role
* @return (*Participant, error)
**/
func (s *Conversation) addParticipant(userId, to, name string, role Role) (*Participant, error) {
	now := timezone.Now()
	participant := newParticipant(s.ia, s.ID, userId, to, name, role)
	participant.JoinedAt = now
	participant.ConversationID = s.ID

	s.mu.Lock()
	defer s.mu.Unlock()

	s.Participants[participant.To] = participant

	return participant, s.save()
}

/**
* addMember
* @param userId, to, name string
* @return (*Participant, error)
**/
func (s *Conversation) addMember(userId, to, name string) (*Participant, error) {
	return s.addParticipant(userId, to, name, Member)
}

/**
* removeParticipant
* @param to string
* @return error
**/
func (s *Conversation) removeParticipant(to string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Participants, to)
	return s.save()
}

/**
* getParticipant
* @param to string
* @return (*Participant, bool)
**/
func (s *Conversation) getParticipant(to string) (*Participant, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	participant, exists := s.Participants[to]
	if !exists {
		return nil, false
	}

	return participant, true
}

/**
* setMessage
* @param to string, tp TypeMessage, content string
* @return error
**/
func (s *Conversation) setMessage(to string, tp TypeMessage, content string) (et.Item, error) {
	participant, exists := s.getParticipant(to)
	if !exists {
		return et.Item{}, fmt.Errorf(MSG_PARTICIPANT_NOT_FOUND)
	}

	ms := newMessage(s.ia, s.ID, participant.UserID, tp, content)
	ms.setStatus(participant.UserID, Sent)
	s.Messages = append(s.Messages, ms)
	s.LastMessage = ms
	if s.ia.store != nil {
		err := s.ia.store.Set(ms.ID, "messages", ms)
		if err != nil {
			return et.Item{}, err
		}
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

/**
* getConversation
* @param id string, tp TypeConversation
* @return (*Conversation, error)
**/
func (s *Conversations) getConversation(id string, tp TypeConversation) (*Conversation, error) {
	if id == "" {
		id = reg.ULID()
	}

	var result *Conversation
	if s.store != nil {
		exists, err := s.store.Get(id, &result)
		if err != nil {
			return nil, err
		}
		if exists {
			result.owner = s
			return result, nil
		}
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

	if s.store != nil {
		err := s.store.Set(id, "conversations", resut)
		if err != nil {
			return nil, err
		}
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
	if s.store != nil {
		exists, err := s.store.Get(id, &result)
		if err != nil {
			return result, err
		}
		if exists {
			return result, nil
		}
	}

	now := timezone.Now()
	result = et.Json{
		"created_at": now,
		"updated_at": now,
		"id":         id,
		"phone":      phone,
	}

	err := s.store.Set(id, s.participantPrefix, result)
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

	if s.store != nil {
		exists, err := s.store.Get(messageId, &ms)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("message not found")
		}
	}

	ms.setStatus(userId, status)
	return s.store.Set(messageId, "messages", ms)
}
