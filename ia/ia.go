package ia

import (
	"context"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/openai/openai-go/v3"
)

var (
	packageName = "ia"
	ia          *Ia
)

type Ia struct {
	ID                string                   `json:"id"`
	Agents            map[string]*Agent        `json:"agents"`
	Conversations     map[string]*Conversation `json:"-"`
	db                *jsql.DB                 `json:"-"`
	sender            Sender                   `json:"-"`
	muAgents          sync.RWMutex             `json:"-"`
	muConversations   sync.RWMutex             `json:"-"`
	key               string                   `json:"-"`
	store             instances.Store          `json:"-"`
	conversationStore instances.Store          `json:"-"`
	messageStore      instances.Store          `json:"-"`
	isDebug           bool                     `json:"-"`
}

/**
* New
* @param db *jsql.DB
* @return (*Ia, error)
**/
func New(db *jsql.DB) (*Ia, error) {
	err := event.Load()
	if err != nil {
		return nil, err
	}

	key := envar.GetStr("OPENAI_API_KEY", "")
	result := &Ia{
		ID:              "ia:agents",
		Agents:          make(map[string]*Agent, 0),
		Conversations:   make(map[string]*Conversation, 0),
		muAgents:        sync.RWMutex{},
		muConversations: sync.RWMutex{},
		isDebug:         envar.GetBool("DEBUG", false),
		key:             key,
		db:              db,
	}
	err = result.up()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* New
* @param db *jsql.DB
* @return error
**/
func Load(db *jsql.DB) error {
	if ia != nil {
		return nil
	}

	var err error
	ia, err = New(db)
	if err != nil {
		return err
	}

	return nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Ia) ToJson() et.Json {
	return et.Json{
		"id":            s.ID,
		"agents":        s.Agents,
		"conversations": s.Conversations,
	}
}

/**
* save
* @return error
**/
func (s *Ia) save() error {
	data := s.ToJson()
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.store != nil {
		err := s.store.Set(s.ID, packageName, "", s)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_IA_SET, data)
	return nil
}

/**
* delete
* @return error
**/
func (s *Ia) delete() error {
	if s.store != nil {
		err := s.store.Delete(s.ID)
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
* up
**/
func (s *Ia) up() error {
	err := s.initStore()
	if err != nil {
		return err
	}

	err = s.initStoreConversation()
	if err != nil {
		return err
	}

	err = s.initStoreMessage()
	if err != nil {
		return err
	}

	err = s.loadAgents()
	if err != nil {
		return err
	}

	err = s.loadConversations()
	if err != nil {
		return err
	}

	return nil
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
* @param name string
* @return (*Agent, error)
**/
func (s *Ia) getAgent(name string) (*Agent, bool) {
	id := agendId(name)
	s.muAgents.RLock()
	result, exists := s.Agents[id]
	s.muAgents.RUnlock()
	if exists {
		return result, true
	}

	if s.store != nil {
		exists, err := s.store.Get(name, &result)
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
func (s *Ia) removeAgent(name string) error {
	id := agendId(name)
	s.muAgents.Lock()
	defer s.muAgents.Unlock()

	delete(s.Agents, id)
	return s.save()
}

/**
* newAgent
* @param name, description, context, model string
* @return (*Agent, error)
**/
func (s *Ia) newAgent(name, description, context, model string) (*Agent, error) {
	_, exists := s.getAgent(name)
	if exists {
		return nil, fmt.Errorf(MSG_AGENT_ALREADY_EXISTS, name)
	}

	result := newAgent(s, name, description, context, model)
	s.addAgent(result)
	return result, s.save()
}

/**
* setModelAgent
* @param agentName string, model string
* @return (*Agent, error)
**/
func (s *Ia) setModelAgent(agentName string, model string) (*Agent, error) {
	if !utility.ValidStr(agentName, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "agentName")
	}
	if !utility.ValidStr(model, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "model")
	}

	result, exists := s.getAgent(agentName)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, agentName)
	}
	result.setModel(model)
	return result, s.save()
}

/**
* setContextAgent
* @param agentName string, context string
* @return (*Agent, error)
**/
func (s *Ia) setContextAgent(agentName string, context string) (*Agent, error) {
	if !utility.ValidStr(agentName, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "agentName")
	}
	if !utility.ValidStr(context, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "context")
	}

	result, exists := s.getAgent(agentName)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, agentName)
	}
	result.setContext(context)
	return result, s.save()
}

/**
* setSkillAgent
* @param agentName string, skill Skill
* @return (*Agent, error)
**/
func (s *Ia) setSkillAgent(agentName string, skill Skill) (*Agent, error) {
	if !utility.ValidStr(agentName, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "agentName")
	}
	if !utility.ValidStr(skill.Tag(), 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "skill")
	}

	result, exists := s.getAgent(agentName)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, agentName)
	}
	result.addSkill(skill)
	return result, s.save()
}

/**
* addConversation
* @param conversation *Conversation
**/
func (s *Ia) addConversation(conversation *Conversation) {
	s.muConversations.Lock()
	defer s.muConversations.Unlock()

	s.Conversations[conversation.ID] = conversation
}

/**
* getConversation
* @param convID string
* @return (*Conversation, bool)
**/
func (s *Ia) getConversation(convID string) (*Conversation, bool) {
	s.muConversations.RLock()
	conversation, exists := s.Conversations[convID]
	s.muConversations.RUnlock()
	if !exists {
		return nil, false
	}

	return conversation, true
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
* @param ctx context.Context, agentName string, convID string, to string, prompt string
* @return (*Conversation, error)
**/
func (s *Ia) Conversation(ctx context.Context, agentName, convID, to, prompt string) (*Conversation, error) {
	if !utility.ValidStr(agentName, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "agentName")
	}
	if !utility.ValidStr(prompt, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "prompt")
	}

	agent, exists := s.getAgent(agentName)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, agentName)
	}

	response, err := agent.conversation(ctx, convID, prompt)
	if err != nil {
		return nil, err
	}

	conversation, exists := s.getConversation(response.ConvID)
	if exists {
		s.addConversation(conversation)
	}

	_, err = conversation.SetTextMessage(to, response.Text)
	if err != nil {
		return nil, err
	}

	return conversation, nil
}
