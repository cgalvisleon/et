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

type Agents struct {
	agents        map[string]*Agent         `json:"-"`
	mu            sync.RWMutex              `json:"-"`
	conversations *Conversations            `json:"-"`
	getInstance   instances.GetInstanceFn   `json:"-"`
	setInstance   instances.SetInstanceFn   `json:"-"`
	queryInstance instances.QueryInstanceFn `json:"-"`
	isDebug       bool                      `json:"-"`
}

/**
* New
* @param store instances.Store
* @return (*Agents, error)
**/
func New(store instances.Store) (*Agents, error) {
	err := event.Load()
	if err != nil {
		return nil, err
	}

	conversations, err := NewConversations("contacts", store)
	if err != nil {
		return nil, err
	}

	result := &Agents{
		agents:        make(map[string]*Agent),
		mu:            sync.RWMutex{},
		conversations: conversations,
		isDebug:       envar.GetBool("DEBUG", false),
	}

	result.getInstance = store.Get
	result.setInstance = store.Set
	result.queryInstance = store.Query
	result.eventInit()

	return result, nil
}

/**
* addAgent
* @param agent *Agent
**/
func (s *Agents) addAgent(agent *Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agents[agent.Tag] = agent
}

/**
* saveAgent
* @param agent *Agent
* @return error
**/
func (s *Agents) saveAgent(agent *Agent) error {
	if s.setInstance != nil {
		return s.setInstance(agent.ID, agent.Tag, agent)
	}

	return nil
}

/**
* newAgent
* @param tag string
* @return *Agent
**/
func (s *Agents) newAgent(tag string) *Agent {
	result, exists := s.Get(tag)
	if exists {
		return result
	}

	id := fmt.Sprintf("ia:%s", tag)
	result = newAgent(s, id, tag)
	result.up()
	result.save()
	s.addAgent(result)

	return result
}

/**
* loadAgent
* @param tag string
* @return (*Agent, error)
**/
func (s *Agents) loadAgent(tag string) (*Agent, error) {
	result, exists := s.Get(tag)
	if exists {
		return result, nil
	}

	if s.getInstance != nil {
		exists, err := s.getInstance(tag, &result)
		if err != nil {
			return nil, err
		}

		if exists {
			result.up()
			s.addAgent(result)
			if s.isDebug {
				logs.Log(packageName, "load:", result.ToString())
			}

			return result, nil
		}
	}

	result = s.newAgent(tag)
	return result, nil
}

/**
* setModel
* @param tag string, model string
* @return (*Agent, error)
**/
func (s *Agents) setModel(tag string, model string) (*Agent, error) {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tag")
	}
	if !utility.ValidStr(model, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "model")
	}

	result, ok := s.agents[tag]
	if !ok {
		return nil, fmt.Errorf(msg.MSG_AGENT_NOT_FOUND, tag)
	}
	result.Model = model
	return result, nil
}

/**
* setContext
* @param tag string, context string
* @return (*Agent, error)
**/
func (s *Agents) setContext(tag string, context string) (*Agent, error) {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tag")
	}
	if !utility.ValidStr(context, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "context")
	}

	result, ok := s.agents[tag]
	if !ok {
		return nil, fmt.Errorf(msg.MSG_AGENT_NOT_FOUND, tag)
	}
	result.Context = context
	return result, nil
}

/**
* Get
* @param tag string
* @return *Agent, bool
**/
func (s *Agents) Get(tag string) (*Agent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result, exists := s.agents[tag]
	return result, exists
}

/**
* Remove
* @param instance *Agent
**/
func (s *Agents) Remove(instance *Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.agents, instance.Tag)
}

/**
* Count
* @return int
**/
func (s *Agents) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.agents)
}

/**
* Load - Carga los agentes
* @param agents []string
* @return error
**/
func (s *Agents) Load(agents []string) error {
	for _, name := range agents {
		_, err := s.loadAgent(name)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* SetContext - Establece el contexto del agente
* @param tag string, context string
* @return error
**/
func (s *Agents) SetContext(tag, context string) error {
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
func (s *Agents) SetModel(tag, model string) error {
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
func (s *Agents) Embed(tag string, text string) ([]float64, error) {
	result, ok := s.agents[tag]
	if !ok {
		return nil, fmt.Errorf(msg.MSG_AGENT_NOT_FOUND, tag)
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
func (s *Agents) Conversations(agent, convID, prompt string) (et.Json, error) {
	if !utility.ValidStr(agent, 1, []string{}) {
		return et.Json{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "agent")
	}

	if !utility.ValidStr(prompt, 1, []string{}) {
		return et.Json{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "prompt")
	}

	ag, ok := s.agents[agent]
	if !ok {
		return et.Json{}, fmt.Errorf(msg.MSG_AGENT_NOT_FOUND, agent)
	}

	response, err := ag.conversations(convID, prompt)
	if err != nil {
		return response, err
	}

	return response, nil

}
