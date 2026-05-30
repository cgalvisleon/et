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
	JoinedAt time.Time `json:"joined_at"`
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	To       string    `json:"to"`
	Name     string    `json:"name"`
	IaID     string    `json:"ia_id"`
	ia       *Ia       `json:"-"`
}

/**
* newParticipant
* @param ia *Ia, userId, to string, role Role
* @return (*Participant, error)
**/
func newParticipant(ia *Ia, userId, to, name string, role Role) (*Participant, error) {
	id := reg.GenULID("participant")
	if userId == "" {
		userId = id
	}
	if name == "" {
		name = to
	}
	result := &Participant{
		JoinedAt: timezone.Now(),
		ID:       id,
		UserID:   userId,
		To:       to,
		Name:     name,
		IaID:     ia.ID,
		ia:       ia,
	}
	err := result.save()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Participant) ToJson() et.Json {
	return et.Json{
		"joined_at": s.JoinedAt,
		"id":        s.ID,
		"user_id":   s.UserID,
		"to":        s.To,
		"name":      s.Name,
		"ia_id":     s.IaID,
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
		err := s.ia.store.Set(s.ID, "participant", s.IaID, s)
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

/**
* SetUserId
* @param userId string
* @return error
**/
func (s *Participant) SetUserId(userId string) error {
	s.UserID = userId
	return s.save()
}

/**
* SetName
* @param name string
* @return error
**/
func (s *Participant) SetName(name string) error {
	s.Name = name
	return s.save()
}
