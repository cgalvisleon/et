package ia

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/openai/openai-go/v3"
)

var (
	packageName = "ia"
	ia          *Ia
)

type Store interface {
	Set(id, tag, tenantId, ownerId string, obj any, userId string) error
	Get(id, tag string, dest any) (bool, error)
	Delete(id, tag string) error
	Query(query et.Json) (et.Items, error)
}

type Config interface {
	GetStr(key string, def string) string
	GetInt(key string, def int) int
	GetBool(key string, def bool) bool
}

type Ia struct {
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
	TenantID      string                   `json:"tenant_id"`
	ID            string                   `json:"id"`
	Tag           string                   `json:"tag"`
	Agents        map[string]*Agent        `json:"agents"`
	Participants  map[string]*Participant  `json:"participants"`
	Conversations map[string]*Conversation `json:"-"`
	sender        Sender                   `json:"-"`
	mutex         map[string]*sync.RWMutex `json:"-"`
	key           string                   `json:"-"`
	store         Store                    `json:"-"`
	isDebug       bool                     `json:"-"`
}

/**
* New
* @param tenantId, tag string, store Store
* @return (*Ia, error)
**/
func New(tenantId, tag string, store Store, config Config) (*Ia, error) {
	err := event.Load(config)
	if err != nil {
		return nil, err
	}

	now := timezone.Now()
	key := envar.GetStr("OPENAI_API_KEY", "")
	isDebug := envar.GetBool("DEBUG", true)
	if config != nil {
		key = config.GetStr("OPENAI_API_KEY", key)
		isDebug = config.GetBool("DEBUG", isDebug)
	}
	result := &Ia{
		CreatedAt:     now,
		UpdatedAt:     now,
		TenantID:      tenantId,
		ID:            fmt.Sprintf("ia:%s", tag),
		Tag:           tag,
		Agents:        make(map[string]*Agent, 0),
		Participants:  make(map[string]*Participant, 0),
		Conversations: make(map[string]*Conversation, 0),
		mutex:         make(map[string]*sync.RWMutex, 0),
		isDebug:       isDebug,
		store:         store,
		key:           key,
	}
	err = result.up()
	if err != nil {
		return nil, err
	}
	result.mutex["agents"] = &sync.RWMutex{}
	result.mutex["participants"] = &sync.RWMutex{}
	result.mutex["conversations"] = &sync.RWMutex{}

	return result, nil
}

/**
* New
* @param tenantId, tag string, store Store, config Config
* @return error
**/
func Load(tenantId, tag string, store Store, config Config) error {
	if ia != nil {
		return nil
	}

	var err error
	ia, err = New(tenantId, tag, store, config)
	if err != nil {
		return err
	}

	return nil
}

/**
* save
* @param userId string
* @return error
**/
func (s *Ia) save(userId string) error {
	s.UpdatedAt = timezone.Now()
	data := s.ToJson()
	data.Set("user_id", userId)
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	event.Publish(EVENT_IA_SET, data)

	if s.store != nil {
		err := s.store.Set(s.ID, packageName, s.TenantID, s.ID, s, userId)
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
	if s.store != nil {
		err := s.store.Delete(s.ID, packageName)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_IA_DELETE, et.Json{
		"id": s.ID,
	})

	return nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Ia) ToJson() et.Json {
	return et.Json{
		"created_at":    timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":    timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"tenant_id":     s.TenantID,
		"owner_id":      s.ID,
		"id":            s.ID,
		"tag":           s.Tag,
		"agents":        s.Agents,
		"conversations": s.Conversations,
	}
}

/**
* up
**/
func (s *Ia) up() error {
	err := s.loadAgents()
	if err != nil {
		return err
	}

	return nil
}

/**
* loadAgents
* @return error
**/
func (s *Ia) loadAgents() error {

	return nil
}

/**
* addAgent
* @param agent *Agent
**/
func (s *Ia) addAgent(agent *Agent) {
	s.mutex["agents"].Lock()
	defer s.mutex["agents"].Unlock()

	s.Agents[agent.ID] = agent
}

/**
* getAgent
* @param tag string
* @return (*Agent, bool)
**/
func (s *Ia) getAgent(tag string) (*Agent, bool) {
	id := agendId(tag)
	s.mutex["agents"].RLock()
	result, exists := s.Agents[id]
	s.mutex["agents"].RUnlock()
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

	result := newAgent(s, tag, name, description, context, model)
	s.addAgent(result)
	return result, s.save(userId)
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
		return nil, fmt.Errorf(MSG_PARTICIPANT_NOT_FOUND)
	}

	result.up(s)
	return result, nil
}

/**
* getParticipant
* @param to, name string, role Role, userId string
* @return (*Participant, error)
**/
func (s *Ia) getParticipant(to, name string, role Role, userId string) (*Participant, error) {
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
		result = newParticipant(s, "", to, name)
		err = result.save(userId)
		if err != nil {
			return nil, err
		}
	}

	s.mutex["participants"].Lock()
	s.Participants[to] = result
	s.mutex["participants"].Unlock()

	return result, s.save(userId)
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
		return nil, fmt.Errorf(MSG_CONVERSATION_NOT_FOUND)
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

	participant, err := s.getParticipant(to, to, Member, userId)
	if err != nil {
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
