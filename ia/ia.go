package ia

import (
	"context"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/openai/openai-go/v3"
)

const (
	packageName = "ia"
)

type Ia struct {
	ID              string                   `json:"id"`
	Agents          map[string]*Agent        `json:"agents"`
	Conversations   map[string]*Conversation `json:"-"`
	muAgents        sync.RWMutex             `json:"-"`
	muConversations sync.RWMutex             `json:"-"`
	key             string                   `json:"-"`
	store           instances.Store          `json:"-"`
	isDebug         bool                     `json:"-"`
}

var ia *Ia

/**
* New
* @param store instances.Store
* @return error
**/
func Load(store instances.Store) error {
	if ia != nil {
		return nil
	}

	err := event.Load()
	if err != nil {
		return err
	}

	key := envar.GetStr("OPENAI_API_KEY", "")
	ia = &Ia{
		ID:              "ia:agents",
		Agents:          make(map[string]*Agent, 0),
		Conversations:   make(map[string]*Conversation, 0),
		muAgents:        sync.RWMutex{},
		muConversations: sync.RWMutex{},
		isDebug:         envar.GetBool("DEBUG", false),
		key:             key,
		store:           store,
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
		err := s.store.Set(s.ID, packageName, s)
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
**/
func (s *Ia) removeAgent(name string) {
	id := agendId(name)
	s.muAgents.Lock()
	defer s.muAgents.Unlock()

	delete(s.Agents, id)
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
	return result, nil
}

/**
* setModel
* @param agentName string, model string
* @return (*Agent, error)
**/
func (s *Ia) setModel(agentName string, model string) (*Agent, error) {
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
	return result, result.save()
}

/**
* setContext
* @param agentName string, context string
* @return (*Agent, error)
**/
func (s *Ia) setContext(agentName string, context string) (*Agent, error) {
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
	return result, result.save()
}

/**
* setSkill
* @param agentName string, skill Skill
* @return (*Agent, error)
**/
func (s *Ia) setSkill(agentName string, skill Skill) (*Agent, error) {
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
	return result, result.save()
}

/**
* SetContext - Establece el contexto del agente
* @param tag string, context string
* @return error
**/
func (s *Ia) SetContext(tag, context string) error {
	result, err := s.setContext(tag, context)
	if err != nil {
		return err
	}

	event.Publish(EVENT_AGENT_SET_CONTEXT, et.Json{
		"tag":     tag,
		"context": context,
	})
	return result.save()
}

/**
* SetModel - Establece el modelo del agente
* @param tag string, model string
* @return error
**/
func (s *Ia) SetModel(tag, model string) error {
	result, err := s.setModel(tag, model)
	if err != nil {
		return err
	}

	event.Publish(EVENT_AGENT_SET_MODEL, et.Json{
		"tag":   tag,
		"model": model,
	})
	return result.save()
}

/**
* Embed - Genera un embedding
* @param tag string, text string
* @return ([]float64, error)
**/
func (s *Ia) Embed(agent string, text string) ([]float64, error) {
	result, ok := s.agents[tag]
	if !ok {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, tag)
	}

	ctx := context.Background()
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
* Conversations
* @param agent string, convID string, prompt string
* @return (et.Json, error)
**/
func (s *Ia) Conversations(agent, convID, prompt string) (et.Json, error) {
	if !utility.ValidStr(agent, 1, []string{}) {
		return et.Json{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "agent")
	}

	if !utility.ValidStr(prompt, 1, []string{}) {
		return et.Json{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "prompt")
	}

	ag, ok := s.agents[agent]
	if !ok {
		return et.Json{}, fmt.Errorf(MSG_AGENT_NOT_FOUND, agent)
	}

	response, err := ag.conversations(convID, prompt)
	if err != nil {
		return response, err
	}

	return response, nil

}
