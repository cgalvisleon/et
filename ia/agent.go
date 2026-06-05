package ia

import (
	"context"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
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
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	ID          string           `json:"id"`
	Tag         string           `json:"tag"`
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
* @param tag string
* @return string
**/
func agendId(tag string) string {
	tag = utility.Normalize(tag)
	return fmt.Sprintf("agent:%s", tag)
}

/**
* newAgent
* @param owner *Ia, tag string, name string, description string, context string, model string
* @return *Agent
**/
func newAgent(ia *Ia, tag, name, description, context, model string) *Agent {
	if context == "" {
		context = contextDefault
	}
	if model == "" {
		model = modelDefault
	}
	now := timezone.Now()
	result := &Agent{
		CreatedAt:   now,
		UpdatedAt:   now,
		ID:          agendId(tag),
		Tag:         tag,
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
* save
* @param userId string
* @return error
**/
func (s *Agent) save(userId string) error {
	s.UpdatedAt = timezone.Now()
	data := s.ToJson()
	data.Set("user_id", userId)
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	event.Publish(EVENT_AGENT_SET, data)

	if s.ia.store != nil {
		return s.ia.store.Set(s.ID, "agent", s.ia.TenantID, s.ia.ID, s, userId)
	}

	return nil
}

/**
* delete
* @return error
**/
func (s *Agent) delete() error {
	if s.ia != nil && s.ia.store != nil {
		err := s.ia.store.Delete(s.ID, "agent")
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_CONVERSATION_DELETE, et.Json{
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
		"created_at":  timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":  timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"tenant_id":   s.ia.TenantID,
		"owner_id":    s.ia.ID,
		"id":          s.ID,
		"tag":         s.Tag,
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
func (s *Agent) Debug() *Agent {
	s.isDebug = true
	return s
}

/**
* setModel
* @param model, userId string
* @return *Agent, error
**/
func (s *Agent) setModel(model, userId string) (*Agent, error) {
	s.Model = model
	return s, s.save(userId)
}

/**
* setContext
* @param context, userId string
* @return *Agent, error
**/
func (s *Agent) setContext(context string, userId string) (*Agent, error) {
	context = strs.Parse(s.ContextBase, et.Json{"context": context})
	s.Context = []byte(context)
	return s, s.save(userId)
}

/**
* addSkill
* @param skill Skill, userId string
* @return *Agent, error
**/
func (s *Agent) addSkill(skill Skill, userId string) (*Agent, error) {
	s.Skills[skill.Tag()] = skill
	return s, s.save(userId)
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
		conversation.SetConvId(convID, s.ID)
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
