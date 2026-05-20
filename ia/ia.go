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

type Ia struct {
	agents          map[string]*Agent `json:"-"`
	conversations   []*Conversation   `json:"-"`
	muAgents        sync.RWMutex      `json:"-"`
	muConversations sync.RWMutex      `json:"-"`
	store           instances.Store   `json:"-"`
	isDebug         bool              `json:"-"`
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

	ia = &Ia{
		agents:          make(map[string]*Agent, 0),
		conversations:   make([]*Conversation, 0),
		muAgents:        sync.RWMutex{},
		muConversations: sync.RWMutex{},
		isDebug:         envar.GetBool("DEBUG", false),
		store:           store,
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

	s.agents[agent.Name] = agent
}

/**
* getAgent
* @param name string
* @return (*Agent, error)
**/
func (s *Ia) getAgent(name string) (*Agent, error) {
	s.muAgents.RLock()
	result, exists := s.agents[name]
	s.muAgents.RUnlock()

	if exists {
		return result, nil
	}

	if s.store != nil {
		exists, err := s.store.Get(name, &result)
		if err != nil {
			return nil, err
		}

		if exists {
			return result, nil
		}
	}

	id := fmt.Sprintf("ia:%s", name)
	result = newAgent(s, id, name)
	result.up()
	result.save()
	s.addAgent(result)

	return result
}

/**
* removeAgent
* @param tag string
* @return (*Agent, error)
**/
func (s *Ia) removeAgent(name string) (*Agent, error) {
	result, exists := s.Get(tag)
	if exists {
		return result, nil
	}

	if s.store != nil {
		exists, err := s.store.Get(tag, &result)
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
func (s *Ia) setModel(tag string, model string) (*Agent, error) {
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
func (s *Ia) setContext(tag string, context string) (*Agent, error) {
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
func (s *Ia) Get(tag string) (*Agent, bool) {
	s.muAgents.RLock()
	defer s.muAgents.RUnlock()

	result, exists := s.agents[tag]
	return result, exists
}

/**
* Remove
* @param instance *Agent
**/
func (s *Ia) Remove(instance *Agent) {
	s.muAgents.Lock()
	defer s.muAgents.Unlock()

	delete(s.agents, instance.Name)
}

/**
* Count
* @return int
**/
func (s *Ia) Count() int {
	s.muAgents.Lock()
	defer s.muAgents.Unlock()

	return len(s.agents)
}

/**
* Load - Carga los agentes
* @param agents []string
* @return error
**/
func (s *Ia) Load(agents []string) error {
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
func (s *Ia) Embed(tag string, text string) ([]float64, error) {
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
func (s *Ia) Conversations(agent, convID, prompt string) (et.Json, error) {
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
