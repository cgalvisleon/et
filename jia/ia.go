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
		ID:            reg.ULID(),
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

	err := s.loadAgents()
	if err != nil {
		return nil, err
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
	for id, agent := range s.Agents {
		agents[id] = agent.ToJson()
	}
	participants := et.Json{}
	for id, participant := range s.Participants {
		participants[id] = participant.ToJson()
	}
	conversations := et.Json{}
	for id, conversation := range s.Conversations {
		conversations[id] = conversation.ToJson()
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
* @param agent *Agent
**/
func (s *Ia) addAgent(agent *Agent) {
	s.muAgents.Lock()
	defer s.muAgents.Unlock()

	s.Agents[agent.ID] = agent
}

/**
* getAgent
* @param tag string
* @return (*Agent, bool)
**/
func (s *Ia) getAgent(id string) (*Agent, bool) {
	s.muAgents.RLock()
	result, exists := s.Agents[id]
	s.muAgents.RUnlock()
	if exists {
		return result, true
	}

	if s.store != nil {
		exists, err := s.store.Get(id, "agent", &result)
		if err != nil {
			return nil, false
		}

		if exists {
			result.up(s)
			s.addAgent(result)
			return result, true
		}
	}

	return nil, false
}

/**
* removeAgent
* @param tag string
* @return error
**/
func (s *Ia) removeAgent(tag, userId string) error {
	id := agendId(tag)
	s.mutex["agents"].Lock()
	defer s.mutex["agents"].Unlock()

	delete(s.Agents, id)
	s.addAuditLog(userId, fmt.Sprintf("remove_agent:%s", tag))
	return s.save(userId)
}

/**
* newAgent
* @param tag, name, description, context, model string
* @return (*Agent, error)
**/
func (s *Ia) newAgent(tag, name, description, context, model, userId string) (*Agent, error) {
	_, exists := s.getAgent(name)
	if exists {
		return nil, fmt.Errorf(MSG_AGENT_ALREADY_EXISTS, name)
	}

	result := newAgent(s, tag, name, description, context, model, userId)
	s.addAgent(result)
	s.addAuditLog(userId, fmt.Sprintf("new_agent:%s", name))
	return result, s.save(userId)
}

/**
* deleteAgent
* @param tag, userId string
* @return error
**/
func (s *Ia) deleteAgent(tag, userId string) error {
	agent, exists := s.getAgent(tag)
	if !exists {
		return fmt.Errorf(MSG_AGENT_NOT_FOUND, tag)
	}

	err := agent.delete()
	if err != nil {
		return err
	}

	return s.removeAgent(tag, userId)
}

/**
* SetModelAgent
* @param name string, model string
* @return *Agent
**/
func (s *Ia) SetModelAgent(name string, model string) (*Agent, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}
	if !utility.ValidStr(model, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "model")
	}

	result, exists := s.getAgent(name)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, name)
	}

	return result.setModel(model), nil
}

/**
* setContextAgent
* @param name string, context string
* @return (*Agent, error)
**/
func (s *Ia) SetContextAgent(name string, context string) (*Agent, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}
	if !utility.ValidStr(context, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "context")
	}

	result, exists := s.getAgent(name)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, name)
	}

	return result.setContext(context), nil
}

/**
* SetSkillAgent
* @param name string, skill Skill
* @return (*Agent, error)
**/
func (s *Ia) SetSkillAgent(name string, skill Skill) (*Agent, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}
	if !utility.ValidStr(skill.Tag(), 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "skill")
	}

	result, exists := s.getAgent(name)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, name)
	}
	return result.addSkill(skill), nil
}

/**
* loadParticipant
* @param to string, dest any
* @return (*Participant, error)
**/
func (s *Ia) loadParticipant(to string) (*Participant, error) {
	var result *Participant
	exists, err := s.store.Get(to, "participant", &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(MSG_PARTICIPANT_NOT_FOUND)
	}

	result.up(s)
	return result, nil
}

var ErrParticipantNotFound = errors.New(MSG_PARTICIPANT_NOT_FOUND)

/**
* getParticipant
* @param to, userId string
* @return (*Participant, error)
**/
func (s *Ia) getParticipant(to, userId string) (*Participant, error) {
	s.mutex["participants"].Lock()
	result, exists := s.Participants[to]
	s.mutex["participants"].Unlock()
	if exists {
		return result, nil
	}

	result, err := s.loadParticipant(to)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, ErrParticipantNotFound
	}

	s.mutex["participants"].Lock()
	s.Participants[to] = result
	s.mutex["participants"].Unlock()

	s.addAuditLog(userId, fmt.Sprintf("load_participant:%s", to))
	return result, s.save(userId)
}

/**
* newParticipant
* @param to, name, userId string
* @return (*Participant, error)
**/
func (s *Ia) newParticipant(to, id, name, userId string) (*Participant, error) {
	result := newParticipant(s, id, to, name)
	err := result.save(userId)
	if err != nil {
		return nil, err
	}

	s.mutex["participants"].Lock()
	s.Participants[to] = result
	s.mutex["participants"].Unlock()

	s.addAuditLog(userId, fmt.Sprintf("new_participant:%s", to))
	return result, s.save(userId)
}

/**
* removeConversation
* @param to, userId string
* @return error
**/
func (s *Ia) removeConversation(to, userId string) error {
	s.mutex["conversations"].Lock()
	defer s.mutex["conversations"].Unlock()

	delete(s.Conversations, to)
	s.addAuditLog(userId, fmt.Sprintf("remove_conversation:%s", to))
	return s.save(userId)
}

/**
* deleteParticipant
* @param to, userId string
* @return error
**/
func (s *Ia) deleteParticipant(to, userId string) error {
	s.mutex["participants"].Lock()
	result, exists := s.Participants[to]
	s.mutex["participants"].Unlock()
	if !exists {
		return fmt.Errorf(MSG_PARTICIPANT_NOT_FOUND, to)
	}

	err := result.delete()
	if err != nil {
		return err
	}

	delete(s.Participants, to)
	return s.save(userId)
}

/**
* loadConversation
* @param to string, dest *Conversation
* @return (bool, error)
**/
func (s *Ia) loadConversation(to *Participant) (*Conversation, error) {
	var result *Conversation
	exists, err := s.store.Get(to.To, "conversation", &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(MSG_CONVERSATION_NOT_FOUND)
	}

	result.up(to)
	return result, nil
}

/**
* getConversation
* @param to *Participant
* @return (*Conversation, error)
**/
func (s *Ia) getConversation(to *Participant, userId string) (*Conversation, error) {
	s.mutex["conversations"].RLock()
	result, exists := s.Conversations[to.To]
	s.mutex["conversations"].RUnlock()
	if !exists {
		return result, nil
	}

	result, err := s.loadConversation(to)
	if err != nil {
		return nil, err
	}

	if !exists {
		result = newConversation(to, to.Name, Direct)
		result.AddParticipant(to)
		err = result.save(userId)
		if err != nil {
			return nil, err
		}
	}

	s.mutex["conversations"].Lock()
	s.Conversations[to.To] = result
	s.mutex["conversations"].Unlock()

	return result, nil
}

/**
* deleteConversation
* @param to, userId string
* @return error
**/
func (s *Ia) deleteConversation(to, userId string) error {
	s.mutex["conversations"].RLock()
	result, exists := s.Conversations[to]
	s.mutex["conversations"].RUnlock()
	if !exists {
		return fmt.Errorf(MSG_CONVERSATION_NOT_FOUND, to)
	}

	err := result.delete()
	if err != nil {
		return err
	}

	return s.removeConversation(to, userId)
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
* @param ctx context.Context, agentName string, convID string, to string, prompt string, userId string
* @return *Conversation, error
**/
func (s *Ia) Conversation(ctx context.Context, tagAgent, to, prompt, userId string) (*Conversation, error) {
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

	participant, err := s.getParticipant(to, userId)
	if errors.Is(err, ErrParticipantNotFound) {
		participant, err = s.newParticipant(to, to, to, userId)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	conversation, err := s.getConversation(participant, userId)
	if err != nil {
		return nil, err
	}

	response, err := agent.conversation(ctx, conversation, prompt)
	if err != nil {
		return nil, err
	}

	_, err = conversation.SendTextMessage(response.Text, userId)
	if err != nil {
		return nil, err
	}

	return conversation, nil
}
