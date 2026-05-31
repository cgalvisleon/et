package ia

import (
	"context"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/conversations"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

const contextDefault = `Eres un asistente que SOLO puede responder con base en el CONTEXTO dado.
{{context}}

Reglas obligatorias:
1. Usa únicamente información del CONTEXTO.
2. No completes con conocimiento externo.
3. No hagas suposiciones.
4. Si la respuesta no está explícitamente en el CONTEXTO, responde exactamente:
"No tengo suficiente información para responder a tu pregunta."
`

const modelDefault = openai.ChatModelGPT4oMini

type Agent struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	ContextBase string           `json:"context_base"`
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
	if context == "" {
		context = contextDefault
	}
	if model == "" {
		model = modelDefault
	}
	result := &Agent{
		ID:          agendId(name),
		Name:        name,
		Description: description,
		ContextBase: context,
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
	context = strs.Parse(s.ContextBase, et.Json{"context": context})
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

type ConversationResult struct {
	ConvID string `json:"conv_id"`
	Text   string `json:"text"`
	Error  string `json:"error"`
}

/**
* ToJson
* @return et.Json
**/
func (s *ConversationResult) ToJson() et.Json {
	return et.Json{
		"conv_id": s.ConvID,
		"text":    s.Text,
		"error":   s.Error,
	}
}

/**
* conversation
* @param ctx context.Context, conversation *Conversation, prompt string
* @return (ConversationResult, error)
**/
func (s *Agent) conversation(ctx context.Context, conversation *Conversation, prompt string) (ConversationResult, error) {
	convID := conversation.ConvID
	if convID == "" {
		conv, _ := s.client.Conversations.New(ctx, conversations.ConversationNewParams{})
		convID = conv.ID
		conversation.SetConvId(convID)
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
		return ConversationResult{
			ConvID: convID,
			Error:  err.Error(),
		}, err
	}

	return ConversationResult{
		ConvID: convID,
		Text:   result.OutputText(),
	}, nil
}
