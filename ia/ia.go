package ia

import (
	"context"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/conversations"
)

type GetInstanceFn func(id string, dest any) (bool, error)
type SetInstanceFn func(id, tag string, obj any) error

type Agents struct {
	agents      map[string]*Agent
	mu          sync.RWMutex
	getInstance GetInstanceFn
	setInstance SetInstanceFn
	isDebug     bool
}

/**
* add - Agrega un agente
* @param agent *Agent
**/
func (s *Agents) add(agent *Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agents[agent.Tag] = agent
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
* remove
* @param instance *Agent
**/
func (s *Agents) remove(instance *Agent) {
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
* @param tag string, context string, model string
* @return *Agent
**/
func (s *Agents) new(tag string) *Agent {
	result := &Agent{
		ID:      fmt.Sprintf("agent:%s", strs.Lowcase(tag)),
		Tag:     "ia:agent",
		Context: contextDefault,
		Model:   openai.ChatModelGPT4oMini,
		isDebug: s.isDebug,
		owner:   s,
	}
	result.Up()
	s.add(result)

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
		s.add(result)

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
		for _, tag := range agents {
			_, exist := s.load(tag)
			if exist {
				continue
			}

			s.new(tag)
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
func (s *Agents) Ask(tag, convID, prompt string) (string, string, error) {
	agent, ok := s.agents[tag]
	if !ok {
		return "", "", fmt.Errorf(msg.MSG_AGENT_NOT_FOUND, tag)
	}

	ctx := context.Background()
	client := agent.client

	if convID == "" {
		conv, _ := client.Conversations.New(ctx, conversations.ConversationNewParams{})
		convID = conv.ID
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(agent.Context),
		openai.UserMessage(prompt),
	}

	result, err := agent.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    agent.Model,
		Messages: messages,
	})
	if err != nil {
		return "", "", err
	}

	return result.Choices[0].Message.Content, convID, nil
}
