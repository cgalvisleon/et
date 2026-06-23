package jia

import (
	"context"
	"errors"
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
	"github.com/openai/openai-go/v3"
)

var (
	packageName = "ia"
)

type Store interface {
	Set(collection, id, ownerId string, obj any) error
	Get(collection, id string, dest any) (bool, error)
	Delete(collection, id string) error
	Query(query et.Json) (et.Items, error)
}

type Ia struct {
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
	ID              string                   `json:"id"`
	Tag             string                   `json:"tag"`
	Agents          map[string]*Agent        `json:"agents"`
	Participants    map[string]*Participant  `json:"-"`
	Conversations   map[string]*Conversation `json:"-"`
	AuditLog        []et.Json                `json:"audit_log"`
	sender          Sender                   `json:"-"`
	muAgents        sync.RWMutex             `json:"-"`
	muParticipants  sync.RWMutex             `json:"-"`
	muConversations sync.RWMutex             `json:"-"`
	key             string                   `json:"-"`
	store           Store                    `json:"-"`
	isDebug         bool                     `json:"-"`
	isChanged       bool                     `json:"-"`
	onSave          []func(ia *Ia) error     `json:"-"`
	onDelete        []func(ia *Ia) error     `json:"-"`
}

/**
* New
* @param tag string, store Store, userId string
* @return (*Ia, error)
**/
func New(tag string, store Store, userId string) (*Ia, error) {
	err := event.Load()
	if err != nil {
		return nil, err
	}

	now := timezone.Now()
	result := &Ia{
		CreatedAt:     now,
		UpdatedAt:     now,
		ID:            reg.UUID(),
		Tag:           tag,
		Agents:        make(map[string]*Agent, 0),
		Participants:  make(map[string]*Participant, 0),
		Conversations: make(map[string]*Conversation, 0),
		AuditLog:      make([]et.Json, 0),
	}
	result.addAuditLog(userId, "new_ia")
	return result.up(store)
}

/**
* New
* @param id string, store Store
* @return error
**/
func Load(id string, store Store) (*Ia, error) {
	if store == nil {
		return nil, errors.New(MSG_STORE_IS_NIL)
	}

	var result *Ia
	exists, err := store.Get(id, packageName, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(MSG_IA_NOT_FOUND)
	}

	return result.up(store)
}

/**
* up
* @param store Store
* @return *Ia, error
**/
func (s *Ia) up(store Store) (*Ia, error) {
	key := envar.GetStr("OPENAI_API_KEY", "")
	isDebug := envar.GetBool("DEBUG", true)
	s.store = store
	s.key = key
	s.isDebug = isDebug
	s.Participants = make(map[string]*Participant, 0)
	s.Conversations = make(map[string]*Conversation, 0)
	s.muAgents = sync.RWMutex{}
	s.muParticipants = sync.RWMutex{}
	s.muConversations = sync.RWMutex{}
	s.onSave = make([]func(ia *Ia) error, 0)
	s.onDelete = make([]func(ia *Ia) error, 0)
	s.OnSave(func(ia *Ia) error {
		event.Publish(EVENT_IA_SET, ia.ToJson())
		return nil
	})
	s.OnDelete(func(ia *Ia) error {
		event.Publish(EVENT_IA_DELETE, et.Json{
			"id": ia.ID,
		})
		return nil
	})

	for id := range s.Agents {
		_, err := s.loadAgend(id)
		if err != nil {
			return nil, err
		}
	}

	return s, nil
}

/**
* addAuditLog
* @param userId string, action string
**/
func (s *Ia) addAuditLog(userId string, action string) {
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
* @param fn func(ia *Ia) error
* @return *Ia
**/
func (s *Ia) OnSave(fn func(ia *Ia) error) *Ia {
	if s.onSave == nil {
		s.onSave = make([]func(ia *Ia) error, 0)
	}
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(ia *Ia) error
* @return *Ia
**/
func (s *Ia) OnDelete(fn func(ia *Ia) error) *Ia {
	if s.onDelete == nil {
		s.onDelete = make([]func(ia *Ia) error, 0)
	}
	s.onDelete = append(s.onDelete, fn)
	return s
}

/**
* save
* @return error
**/
func (s *Ia) save() error {
	if s.store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	s.isChanged = false

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	err := s.store.Set("ia", s.ID, s.ID, s)
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
func (s *Ia) delete() error {
	if s.store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	err := s.store.Delete("ia", s.ID)
	if err != nil {
		return err
	}

	for _, onDelete := range s.onDelete {
		if err := onDelete(s); err != nil {
			return err
		}
	}

	return nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Ia) ToJson() et.Json {
	agents := et.Json{}
	for tag, agent := range s.Agents {
		agents[tag] = agent.ToJson()
	}
	participants := et.Json{}
	for to, participant := range s.Participants {
		participants[to] = participant.ToJson()
	}
	conversations := et.Json{}
	for convID, conversation := range s.Conversations {
		conversations[convID] = conversation.ToJson()
	}
	return et.Json{
		"created_at":    timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":    timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"id":            s.ID,
		"tag":           s.Tag,
		"agents":        agents,
		"participants":  participants,
		"conversations": conversations,
		"audit_log":     s.AuditLog,
	}
}

/**
* ToString
* @return string
**/
func (s *Ia) ToString() string {
	return s.ToJson().ToString()
}

/**
* addAgent
* @param agent *Agent, userId string
**/
func (s *Ia) addAgent(agent *Agent, userId string) {
	s.muAgents.Lock()
	defer s.muAgents.Unlock()

	agent.addAuditLog(userId, "add_agent")
	s.Agents[agent.Tag] = agent
}

/**
* getAgent
* @param tag string
* @return (*Agent, bool)
**/
func (s *Ia) getAgent(tag string) (*Agent, bool) {
	s.muAgents.RLock()
	result, exists := s.Agents[tag]
	s.muAgents.RUnlock()
	if exists {
		return result, true
	}

	return nil, false
}

/**
* removeAgent
* @param tag, userId string
**/
func (s *Ia) removeAgent(tag, userId string) {
	s.muAgents.Lock()
	defer s.muAgents.Unlock()

	s.addAuditLog(userId, "remove_agent")
	delete(s.Agents, tag)
}

/**
* SetModelAgent
* @param id string, model, userId string
* @return *Agent
**/
func (s *Ia) SetModelAgent(tag string, model, userId string) (*Agent, error) {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tag")
	}
	if !utility.ValidStr(model, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "model")
	}

	result, exists := s.getAgent(tag)
	if !exists {
		return nil, errors.New(MSG_AGENT_NOT_FOUND)
	}

	return result.setModel(model, userId), nil
}

/**
* SetContextAgent
* @param tag string, context, userId string
* @return (*Agent, error)
**/
func (s *Ia) SetContextAgent(tag string, context, userId string) (*Agent, error) {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tag")
	}
	if !utility.ValidStr(context, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "context")
	}

	result, exists := s.getAgent(tag)
	if !exists {
		return nil, errors.New(MSG_AGENT_NOT_FOUND)
	}

	return result.setContext(context, userId), nil
}

/**
* SetSkillAgent
* @param tag string, skill Skill, userId string
* @return (*Agent, error)
**/
func (s *Ia) SetSkillAgent(tag string, skill Skill, userId string) (*Agent, error) {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tag")
	}
	if !utility.ValidStr(skill.Tag(), 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "skill")
	}

	result, exists := s.getAgent(tag)
	if !exists {
		return nil, errors.New(MSG_AGENT_NOT_FOUND)
	}
	return result.addSkill(skill, userId), nil
}

/**
* addParticipant
* @param participant *Participant, userId string
**/
func (s *Ia) addParticipant(participant *Participant, userId string) {
	s.muParticipants.Lock()
	defer s.muParticipants.Unlock()

	s.addAuditLog(userId, "add_participant")
	s.Participants[participant.To] = participant
}

/**
* getParticipant
* @param to string
* @return (*Participant, error)
**/
func (s *Ia) getParticipant(to string) (*Participant, bool) {
	s.muParticipants.RLock()
	result, exists := s.Participants[to]
	s.muParticipants.RUnlock()
	if exists {
		return result, true
	}

	return nil, false
}

/**
* removeParticipant
* @param to, userId string
**/
func (s *Ia) removeParticipant(to, userId string) {
	s.muParticipants.Lock()
	defer s.muParticipants.Unlock()

	s.addAuditLog(userId, "remove_participant")
	delete(s.Participants, to)
}

/**
* addConversation
* @param conversation *Conversation, userId string
**/
func (s *Ia) addConversation(conversation *Conversation, userId string) {
	s.muConversations.Lock()
	defer s.muConversations.Unlock()

	s.addAuditLog(userId, "add_conversation")
	s.Conversations[conversation.ConvID] = conversation
}

/**
* getConversation
* @param convID string
* @return (*Conversation, bool)
**/
func (s *Ia) getConversation(convID string) (*Conversation, bool) {
	s.muConversations.RLock()
	result, exists := s.Conversations[convID]
	s.muConversations.RUnlock()
	if exists {
		return result, true
	}

	return nil, false
}

/**
* removeConversation
* @param convID, userId string
**/
func (s *Ia) removeConversation(convID, userId string) {
	s.muConversations.Lock()
	defer s.muConversations.Unlock()

	s.addAuditLog(userId, "remove_conversation")
	delete(s.Conversations, convID)
}

/**
* deleteConversation
* @param convID, userId string
* @return error
**/
func (s *Ia) deleteConversation(convID, userId string) error {
	conversation, exists := s.getConversation(convID)
	if !exists {
		return errors.New(MSG_CONVERSATION_NOT_FOUND)
	}

	err := conversation.delete()
	if err != nil {
		return err
	}

	s.removeConversation(convID, userId)
	return nil
}

/**
* Embed - Genera un embedding
* @param ctx context.Context, agentName string, text string
* @return ([]float64, error)
**/
func (s *Ia) Embed(ctx context.Context, agentName string, text string) ([]float64, error) {
	if !utility.ValidStr(agentName, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "agentName")
	}
	if !utility.ValidStr(text, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "text")
	}

	result, exists := s.getAgent(agentName)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, agentName)
	}

	client := result.client
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModelTextEmbedding3Small,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
	})
	if err != nil {
		return nil, err
	}

	return resp.Data[0].Embedding, nil
}

/**
* Conversation
* @param ctx context.Context, agentName, to, name, prompt string
* @return *Conversation, error
**/
func (s *Ia) Conversation(ctx context.Context, tagAgent, to, name, prompt string) (*Conversation, error) {
	if !utility.ValidStr(tagAgent, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tagAgent")
	}
	if !utility.ValidStr(to, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "to")
	}
	if !utility.ValidStr(prompt, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "prompt")
	}

	agent, exists := s.getAgent(tagAgent)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, tagAgent)
	}

	participant, exists := s.getParticipant(to)
	if !exists {
		var err error
		participant, err = s.newParticipant(to, name, to)
		if err != nil {
			return nil, err
		}
	}

	response, err := agent.conversation(ctx, participant, prompt)
	if err != nil {
		return nil, err
	}

	conversation, exists := s.getConversation(participant.ConvID)
	if !exists {
		conversation = s.newConversation(participant, to, DIRECT)
	}

	_, err = conversation.SendTextMessage(response.Text, participant.ID, to)
	if err != nil {
		return nil, err
	}

	return conversation, nil
}
