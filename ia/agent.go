package ia

import (
	"context"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/conversations"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

const contextDefault = `Eres un asistente que SOLO puede responder con base en el CONTEXTO dado.
{{contexto}}

Reglas obligatorias:
1. Usa únicamente información del CONTEXTO.
2. No completes con conocimiento externo.
3. No hagas suposiciones.
4. Si la respuesta no está explícitamente en el contexto, responde exactamente:
"No tengo suficiente información para responder a tu pregunta."
`

const modelDefault = openai.ChatModelGPT4oMini

type Skill interface {
	Tag() string
	Name() string
	Description() string
	Execute(
		ctx context.Context,
		input map[string]any,
	) (*SkillResult, error)
}

type SkillResult struct {
	Success bool
	Data    any
	Error   string
}

type Agent struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Context     []byte           `json:"context"`
	Model       string           `json:"model"`
	Skills      map[string]Skill `json:"skills"`
	client      openai.Client    `json:"-"`
	ia          *Ia              `json:"-"`
	isDebug     bool             `json:"-"`
}

/**
* agendId
* @param name string
* @return string
**/
func agendId(name string) string {
	name = utility.Normalize(name)
	return fmt.Sprintf("agent:%s", name)
}

/**
* newAgent
* @param owner *Ia, name string, description string, context string, model string
* @return *Agent
**/
func newAgent(ia *Ia, name, description, context, model string) *Agent {
	result := &Agent{
		ID:          agendId(name),
		Name:        name,
		Description: description,
		Context:     []byte(context),
		Skills:      make(map[string]Skill),
		Model:       model,
		ia:          ia,
		isDebug:     ia.isDebug,
	}
	ia.addAgent(result)
	return result
}

/**
* save
* @return error
**/
func (s *Agent) save() error {
	data := s.ToJson()
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.ia != nil && s.ia.store != nil {
		err := s.ia.store.Set(s.ID, "agent", s)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_AGENT_SET, data)

	return nil
}

/**
* delete
* @return error
**/
func (s *Agent) delete() error {
	if s.ia != nil && s.ia.store != nil {
		err := s.ia.store.Delete(s.ID)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_AGENT_DELETE, et.Json{
		"id": s.ID,
	})

	return nil
}

/**
* up
* @param ia *Ia
**/
func (s *Agent) up(ia *Ia) {
	s.client = openai.NewClient(
		option.WithAPIKey(ia.key),
	)
	s.ia = ia
	s.isDebug = ia.isDebug
}

/**
* ToJson
* @return et.Json
**/
func (s *Agent) ToJson() et.Json {
	return et.Json{
		"id":          s.ID,
		"name":        s.Name,
		"description": s.Description,
		"context":     s.Context,
		"model":       s.Model,
		"skills":      s.Skills,
	}
}

/**
* ToString
* @return string
**/
func (s *Agent) ToString() string {
	return s.ToJson().ToString()
}

/**
* Debug
**/
func (s *Agent) Debug() {
	s.isDebug = true
}

/**
* setModel
* @param model string
* @return *Agent
**/
func (s *Agent) setModel(model string) *Agent {
	s.Model = model
	return s
}

/**
* setContext
* @param context string
* @return *Agent
**/
func (s *Agent) setContext(context string) *Agent {
	s.Context = []byte(context)
	return s
}

/**
* addSkill
* @param skill Skill
* @return *Agent
**/
func (s *Agent) addSkill(skill Skill) *Agent {
	s.Skills[skill.Tag()] = skill
	return s
}

/**
* conversations
* @param ctx context.Context, convID, prompt string
* @return (string, string, error)
**/
func (s *Agent) conversations(ctx context.Context, convID, prompt string) (et.Json, error) {
	if convID == "" {
		conv, _ := s.client.Conversations.New(ctx, conversations.ConversationNewParams{})
		convID = conv.ID
	}

	contextStr := string(s.Context)
	prompt = fmt.Sprintf(contextStr, prompt)
	result, err := s.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: s.Model,
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
		return et.Json{
			"conv_id": convID,
		}, err
	}

	return et.Json{
		"conv_id":  convID,
		"response": result.OutputText(),
	}, nil
}
