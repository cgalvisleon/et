package jia

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type Role string

const (
	Admin  Role = "admin"
	Member Role = "member"
)

type Participant struct {
	CreatedAt time.Time                              `json:"created_at"`
	UpdatedAt time.Time                              `json:"updated_at"`
	JoinedAt  time.Time                              `json:"joined_at"`
	IAID      string                                 `json:"ia_id"`
	ID        string                                 `json:"id"`
	To        string                                 `json:"to"`
	Name      string                                 `json:"name"`
	ConvID    string                                 `json:"conv_id"`
	AuditLog  []et.Json                              `json:"audit_log"`
	isDebug   bool                                   `json:"-"`
	isChanged bool                                   `json:"-"`
	ia        *Ia                                    `json:"-"`
	store     Store                                  `json:"-"`
	onSave    []func(participant *Participant) error `json:"-"`
	onDelete  []func(participant *Participant) error `json:"-"`
}

/**
* newParticipant
* @param to, name, userId string
* @return (*Participant, error)
**/
func (s *Ia) newParticipant(to, name, userId string) (*Participant, error) {
	now := timezone.Now()
	id := reg.UUID()
	result := &Participant{
		CreatedAt: now,
		UpdatedAt: now,
		JoinedAt:  now,
		ID:        id,
		To:        to,
		Name:      name,
		ConvID:    "",
	}
	result.up(s)
	s.addParticipant(result, userId)
	return result, nil
}

/**
* loadParticipant
* @param to string
* @return (*Participant, error)
**/
func (s *Ia) loadParticipant(to string) (*Participant, error) {
	var result *Participant
	exists, err := s.store.Get("participant", to, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(MSG_PARTICIPANT_NOT_FOUND)
	}

	result.up(s)
	return result, nil
}

/**
* deleteParticipant
* @param to, userId string
* @return error
**/
func (s *Ia) deleteParticipant(to, userId string) error {
	result, exists := s.getParticipant(to)
	if !exists {
		return errors.New(MSG_PARTICIPANT_NOT_FOUND)
	}

	err := result.delete()
	if err != nil {
		return err
	}

	s.removeParticipant(to, userId)
	return nil
}

/**
* up
* @param ia *Ia
**/
func (s *Participant) up(ia *Ia) {
	s.ia = ia
	s.isDebug = ia.isDebug
	s.store = ia.store
	s.onSave = make([]func(participant *Participant) error, 0)
	s.onDelete = make([]func(participant *Participant) error, 0)
	s.OnSave(func(participant *Participant) error {
		key := fmt.Sprintf("participant:%s", participant.ID)
		event.Publish(key, participant.ToJson())
		return nil
	})
	s.OnDelete(func(participant *Participant) error {
		key := fmt.Sprintf("participant:%s:delete", participant.ID)
		event.Publish(key, et.Json{
			"id": participant.ID,
		})
		return nil
	})
}

/**
* addAuditLog
* @param userId string, action string
**/
func (s *Participant) addAuditLog(userId string, action string) {
	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	s.UpdatedAt = now
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	maxAuditLog := envar.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}
	s.isChanged = true
}

/**
* OnSave
* @param fn func(step *Step, userId string) error
* @return *Participant
**/
func (s *Participant) OnSave(fn func(participant *Participant) error) *Participant {
	if s.onSave == nil {
		s.onSave = make([]func(participant *Participant) error, 0)
	}
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(step *Step, userId string) error
* @return *Step
**/
func (s *Participant) OnDelete(fn func(participant *Participant) error) *Participant {
	if s.onDelete == nil {
		s.onDelete = make([]func(participant *Participant) error, 0)
	}
	s.onDelete = append(s.onDelete, fn)
	return s
}

/**
* save
* @return error
**/
func (s *Participant) save() error {
	if s.store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	s.isChanged = false

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	err := s.store.Set("participant", s.ID, s.IAID, s)
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
* ToJson
* @return et.Json
**/
func (s *Participant) ToJson() et.Json {
	return et.Json{
		"created_at": timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at": timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"joined_at":  timezone.Format(s.JoinedAt, timezone.RFC3339),
		"ia_id":      s.IAID,
		"id":         s.ID,
		"to":         s.To,
		"name":       s.Name,
		"conv_id":    s.ConvID,
		"audit_log":  s.AuditLog,
	}
}

/**
* ToString
* @return string
**/
func (s *Participant) ToString() string {
	return s.ToJson().ToString()
}

/**
* delete
* @return error
**/
func (s *Participant) delete() error {
	if s.store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	err := s.store.Delete("participant", s.ID)
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
* SetName
* @param name, userId string
* @return *Participant
**/
func (s *Participant) SetName(name, userId string) *Participant {
	if s.Name == name {
		return s
	}
	s.Name = name
	s.addAuditLog(userId, "set_name")
	return s
}

/**
* SetConvId
* @param convId, userId string
* @return *Participant
**/
func (s *Participant) SetConvId(convId, userId string) *Participant {
	if s.ConvID == convId {
		return s
	}
	s.ConvID = convId
	s.addAuditLog(userId, "set_conv_id")
	return s
}
