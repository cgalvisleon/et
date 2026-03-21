package ia

import (
	"context"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

var contextDefault = `Responde SOLO usando el contexto.
Si la respuesta no está en el contexto, responde: "No tengo suficiente información para responder a tu pregunta."
Contexto:`

type Agent struct {
	ID      string        `json:"id"`
	Tag     string        `json:"tag"`
	Context string        `json:"context"`
	Model   string        `json:"model"`
	client  openai.Client `json:"-"`
}

func newAgent(tag string, context string, model string) *Agent {
	return &Agent{
		ID:      fmt.Sprintf("agent:%s", strs.Lowcase(tag)),
		Tag:     "ia:agent",
		Context: context,
		Model:   model,
	}
}

/**
* Up
**/
func (s *Agent) Up() {
	key := envar.GetStr("OPENAI_API_KEY", "")
	client := openai.NewClient(
		option.WithAPIKey(key),
	)
	s.client = client
}

type GetInstanceFn func(id string, dest any) (bool, error)
type SetInstanceFn func(id, tag string, obj any) error

type Agents struct {
	agents      map[string]*Agent
	mu          sync.RWMutex
	getInstance GetInstanceFn
	setInstance SetInstanceFn
}

func New(store instances.Store) *Agents {
	result := &Agents{
		agents: make(map[string]*Agent),
		mu:     sync.RWMutex{},
	}

	if store != nil {
		result.getInstance = store.Get
		result.setInstance = store.Set
	}

	return result
}

/**
* save - Guarda el agente
* @return error
**/
func (s *Agents) save(ag *Agent) error {
	if s.setInstance == nil {
		return nil
	}

	return s.setInstance(ag.ID, ag.Tag, ag)
}

/**
* get
* @param tag string
* @return (*Agent, bool)
**/
func (s *Agents) get(tag string) (*Agent, bool) {
	s.mu.RLock()
	result, exists := s.agents[tag]
	s.mu.RUnlock()
	if exists {
		return result, true
	}

	if s.getInstance != nil {
		exists, err := s.getInstance(tag, &result)
		if err != nil {
			return nil, false
		}
		if exists {
			result.Up()
			return result, false
		}
	}

	result = newAgent(tag, contextDefault, openai.ChatModelGPT4oMini)
	result.Up()
	return result, false
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
* Load
* @return error
 */
func (s *Agents) Load(agents []string) error {
	for _, tag := range agents {
		agent, exists := s.get(tag)
		if exists {
			continue
		}

		if agent != nil {
			s.add(agent)
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
	agent, ok := s.agents[tag]
	if !ok {
		return fmt.Errorf(msg.MSG_AGENT_NOT_FOUND, tag)
	}
	agent.Context = context
	return s.save(agent)
}

/**
* SetModel - Establece el modelo del agente
* @param tag string, model string
* @return error
**/
func (s *Agents) SetModel(tag, model string) error {
	agent, ok := s.agents[tag]
	if !ok {
		return fmt.Errorf(msg.MSG_AGENT_NOT_FOUND, tag)
	}
	agent.Model = model
	return s.save(agent)
}

/**
* Ask - Pregunta al agente
* @param tag string, prompt string
* @return (string, error)
**/
func (s *Agents) Ask(tag, prompt string) (string, error) {
	agent, ok := s.agents[tag]
	if !ok {
		return "", fmt.Errorf(msg.MSG_AGENT_NOT_FOUND, tag)
	}

	ctx := context.Background()
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(agent.Context),
		openai.UserMessage(prompt),
	}

	result, err := agent.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    agent.Model,
		Messages: messages,
	})
	if err != nil {
		return "", err
	}

	return result.Choices[0].Message.Content, nil
}
