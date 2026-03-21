package ia

import (
	"context"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/conversations"
	"github.com/openai/openai-go/v3/responses"
)

type Agents struct {
	agents        map[string]*Agent
	mu            sync.RWMutex
	getInstance   instances.GetInstanceFn
	setInstance   instances.SetInstanceFn
	queryInstance instances.QueryInstanceFn
	isDebug       bool
}

/**
* add - Agrega un agente
* @param agent *Agent
**/
func (s *Agents) add(agent *Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agents[agent.ID] = agent
}

/**
* Get
* @param tag string
* @return *Agent, bool
**/
func (s *Agents) Get(tag string) (*Agent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, ok := s.agents[tag]
	return result, ok
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
* new
* @param id string
* @return *Agent
**/
func (s *Agents) new(id string) *Agent {
	if id == "" {
		id = reg.ULID()
	}
	result := &Agent{
		ID:      id,
		Tag:     "ia:agent",
		Context: contextDefault,
		Model:   openai.ChatModelGPT4oMini,
		isDebug: s.isDebug,
		owner:   s,
	}
	result.Up()

	return result
}

/**
* load
* @param tag string
* @return (*Agent, bool)
**/
func (s *Agents) load(tag string) (*Agent, bool) {
	result, exists := s.Get(tag)
	if exists {
		return result, true
	}

	if s.getInstance != nil {
		exists, err := s.getInstance(tag, &result)
		if err != nil {
			return nil, false
		}

		if !exists {
			return nil, false
		}

		result.Up()

		if s.isDebug {
			logs.Log(packageName, "load:", result.ToString())
		}

		return result, true
	}

	return nil, false
}

/**
* Load - Carga los agentes
* @param agents []string
* @return error
**/
func (s *Agents) Load(agents []string) error {
	if s.setInstance != nil {
		for _, name := range agents {
			ag, exist := s.load(name)
			if exist {
				s.add(ag)
				continue
			}

			ag = s.new(name)
			err := ag.Save()
			if err != nil {
				return err
			}
			s.add(ag)
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
	result, ok := s.agents[tag]
	if !ok {
		return fmt.Errorf(msg.MSG_AGENT_NOT_FOUND, tag)
	}
	result.Context = context
	return result.Save()
}

/**
* SetModel - Establece el modelo del agente
* @param tag string, model string
* @return error
**/
func (s *Agents) SetModel(tag, model string) error {
	result, ok := s.agents[tag]
	if !ok {
		return fmt.Errorf(msg.MSG_AGENT_NOT_FOUND, tag)
	}
	result.Model = model
	return result.Save()
}

/**
* Ask - Pregunta al agente
* @param tag string, convID string, prompt string
* @return (string, error)
**/
func (s *Agents) Ask(agent, convID, prompt string) (string, string, error) {
	if !utility.ValidStr(agent, 1, []string{}) {
		return "", "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "agent")
	}

	if !utility.ValidStr(prompt, 1, []string{}) {
		return "", "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "prompt")
	}

	instance, ok := s.agents[agent]
	if !ok {
		return "", "", fmt.Errorf(msg.MSG_AGENT_NOT_FOUND, agent)
	}

	ctx := context.Background()
	client := instance.client

	if convID == "" {
		conv, _ := client.Conversations.New(ctx, conversations.ConversationNewParams{})
		convID = conv.ID

		response, err := client.Responses.New(ctx, responses.ResponseNewParams{
			Model: instance.Model,
			Input: responses.ResponseNewParamsInputUnion{
				OfString: openai.String(instance.Context),
			},
			Conversation: responses.ResponseNewParamsConversationUnion{
				OfConversationObject: &responses.ResponseConversationParam{
					ID: convID,
				},
			},
		})
		if err != nil {
			return "", convID, err
		}
		logs.Log(packageName, "response:", response.OutputText())
	}

	result, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Model: instance.Model,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(prompt),
		},
		Conversation: responses.ResponseNewParamsConversationUnion{
			OfConversationObject: &responses.ResponseConversationParam{
				ID: convID,
			},
		},
	})
	if err != nil {
		return "", convID, err
	}

	return result.OutputText(), convID, nil
}
