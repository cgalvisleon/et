package ia

import (
	"time"

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
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	JoinedAt  time.Time `json:"joined_at"`
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	To        string    `json:"to"`
	Name      string    `json:"name"`
	ia        *Ia       `json:"-"`
	isChanged bool      `json:"-"`
}

/**
* newParticipant
* @param ia *Ia, userId, to string
* @return (*Participant, error)
**/
func newParticipant(ia *Ia, userId, to, name string) *Participant {
	id := reg.GenULID("participant")
	if userId == "" {
		userId = id
	}

	if name == "" {
		name = to
	}

	now := timezone.Now()
	result := &Participant{
		CreatedAt: now,
		UpdatedAt: now,
		JoinedAt:  now,
		ID:        id,
		UserID:    userId,
		To:        to,
		Name:      name,
		ia:        ia,
	}

	return result
}

/**
* save
* @return error
**/
func (s *Participant) save(userId string) error {
	s.UpdatedAt = timezone.Now()
	data := s.ToJson()
	data.Set("user_id", userId)
	if s.ia.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.ia.store != nil {
		err := s.ia.store.Set(s.ID, "participant", s.ia.TenantID, s.ia.ID, s, userId)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_PARTICIPANT_SET, data)

	s.isChanged = false
	return nil
}

/**
* delete
* @return error
**/
func (s *Participant) delete() error {
	if s.ia.store != nil {
		err := s.ia.store.Delete(s.ID, "participant")
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
* ToJson
* @return et.Json
**/
func (s *Participant) ToJson() et.Json {
	return et.Json{
		"created_at": timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at": timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"joined_at":  timezone.Format(s.JoinedAt, timezone.RFC3339),
		"tenant_id":  s.ia.TenantID,
		"owner_id":   s.ia.ID,
		"id":         s.ID,
		"user_id":    s.UserID,
		"to":         s.To,
		"name":       s.Name,
	}
}

/**
* up
* @param ia *Ia
**/
func (s *Participant) up(ia *Ia) {
	s.ia = ia
}

/**
* SetUserId
* @param userId string
* @return error
**/
func (s *Participant) SetUserId(id string) *Participant {
	if s.UserID == id {
		return s
	}
	s.UserID = id
	s.isChanged = true
	return s
}

/**
* SetName
* @param name string
* @return error
**/
func (s *Participant) SetName(name string) *Participant {
	if s.Name == name {
		return s
	}
	s.Name = name
	s.isChanged = true
	return s
}
