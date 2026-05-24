package ia

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type Conversation struct {
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
	ID           string                  `json:"id"`
	Type         TypeConversation        `json:"type"`
	Participants map[string]*Participant `json:"participants"`
	Messages     []*Message              `json:"messages"`
	LastMessage  *Message                `json:"last_message"`
	owner        *Ia                     `json:"-"`
	isDebug      bool                    `json:"-"`
}

func (s *Conversation) ToJson() et.Json {
	return et.Json{
		"id":           s.ID,
		"type":         s.Type,
		"participants": s.Participants,
		"messages":     s.Messages,
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

	if s.owner.store != nil {
		return s.owner.store.Set(s.ID, "conversations", s)
	}

	return nil
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
			ID:             reg.ULID(),
			ConversationID: s.ID,
			UserID:         userId,
			To:             to,
			Role:           Member,
		}

		if s.owner.store != nil {
			err = s.owner.store.Set(userId, "participants", participant)
			if err != nil {
				return et.Item{}, err
			}
		}

		s.Participants[userId] = participant
	}

	ms := newMessage(s.ID, userId, tp, content)
	ms.setStatus(userId, Sent)
	s.Messages = append(s.Messages, ms)
	s.LastMessage = ms
	if s.owner.store != nil {
		err := s.owner.store.Set(ms.ID, "messages", ms)
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

type Conversations struct {
	participantPrefix string          `json:"-"`
	store             instances.Store `json:"-"`
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
		store:             store,
	}

	return result, nil
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
