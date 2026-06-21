package jia

import (
	"context"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/envar"
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
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
	ID          string                     `json:"id"`
	Tag         string                     `json:"tag"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	ContextBase string                     `json:"context_base"`
	Context     []byte                     `json:"context"`
	Model       string                     `json:"model"`
	Skills      map[string]Skill           `json:"skills"`
	AuditLog    []et.Json                  `json:"audit_log"`
	client      openai.Client              `json:"-"`
	ia          *Ia                        `json:"-"`
	isDebug     bool                       `json:"-"`
	isChanged   bool                       `json:"-"`
	onSave      []func(agent *Agent) error `json:"-"`
	onDelete    []func(agent *Agent) error `json:"-"`
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
* @param owner *Ia, tag string, name string, description string, context string, model string, userId string
* @return *Agent
**/
func newAgent(ia *Ia, tag, name, description, context, model, userId string) *Agent {
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
		AuditLog:    make([]et.Json, 0),
		Model:       model,
	}
	result.addAuditLog(userId, "new_agent")
	result.up(ia)
	ia.addAgent(result)
	return result
}

/**
* up
* @param ia *Ia
* @return *Agent
**/
func (s *Agent) up(ia *Ia) *Agent {
	s.client = openai.NewClient(
		option.WithAPIKey(ia.key),
	)
	s.ia = ia
	s.isDebug = ia.isDebug
	s.onSave = make([]func(agent *Agent) error, 0)
	s.onDelete = make([]func(agent *Agent) error, 0)
	s.OnSave(func(agent *Agent) error {
		event.Publish(EVENT_AGENT_SET, agent.ToJson())
		return nil
	})
	s.OnDelete(func(agent *Agent) error {
		event.Publish(EVENT_AGENT_DELETE, et.Json{
			"id": agent.ID,
		})
		return nil
	})
	return s
}

/**
* OnSave
* @param fn func(agent *Agent) error
* @return *Agent
**/
func (s *Agent) OnSave(fn func(agent *Agent) error) *Agent {
	if s.onSave == nil {
		s.onSave = make([]func(agent *Agent) error, 0)
	}
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(agent *Agent) error
* @return *Agent
**/
func (s *Agent) OnDelete(fn func(agent *Agent) error) *Agent {
	if s.onDelete == nil {
		s.onDelete = make([]func(agent *Agent) error, 0)
	}
	s.onDelete = append(s.onDelete, fn)
	return s
}

/**
* addAuditLog
* @param userId string, action string
**/
func (s *Agent) addAuditLog(userId string, action string) {
	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	maxAuditLog := envar.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}
	s.isChanged = true
}

/**
* save
* @param userId string
* @return error
**/
func (s *Agent) save(userId string) error {
	s.UpdatedAt = timezone.Now()
	s.addAuditLog(userId, "update_agent")
	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	if s.ia.store != nil {
		err := s.ia.store.Set(s.ID, "agent", s.ia.TenantID, s.ia.ID, s)
		if err != nil {
			return err
		}
	}

	s.isChanged = false

	for _, onSave := range s.onSave {
		if err := onSave(s); err != nil {
			return err
		}
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

	for _, onDelete := range s.onDelete {
		if err := onDelete(s); err != nil {
			return err
		}
	}

	return nil
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
* @param model string
* @return *Agent, error
**/
func (s *Agent) setModel(model string) *Agent {
	if s.Model == model {
		return s
	}
	s.Model = model
	s.isChanged = true
	return s
}

/**
* setContext
* @param context, userId string
* @return *Agent
**/
func (s *Agent) setContext(context string) *Agent {
	context = strs.Parse(s.ContextBase, et.Json{"context": context})
	if string(s.Context) == context {
		return s
	}
	s.Context = []byte(context)
	s.isChanged = true
	return s
}

/**
* addSkill
* @param skill Skill, userId string
* @return *Agent, error
**/
func (s *Agent) addSkill(skill Skill) *Agent {
	_, ok := s.Skills[skill.Tag()]
	if ok {
		return s
	}
	s.Skills[skill.Tag()] = skill
	s.isChanged = true
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
		conversation.save(s.ID)
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
