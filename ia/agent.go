package ia

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
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

type Agent struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Context string          `json:"context"`
	Model   string          `json:"model"`
	client  openai.Client   `json:"-"`
	ctx     context.Context `json:"-"`
	owner   *Ia             `json:"-"`
	isDebug bool            `json:"-"`
}

/**
* newAgent
* @param ctx context.Context, owner *Ia, name string
* @return *Agent
**/
func newAgent(ctx context.Context, owner *Ia, name string) *Agent {
	return &Agent{
		ID:      fmt.Sprintf("agent:%s", name),
		Name:    name,
		Context: contextDefault,
		Model:   modelDefault,
		ctx:     ctx,
		owner:   owner,
		isDebug: owner.isDebug,
	}
}

/**
* ToJson
* @return (et.Json, error)
**/
func (s *Agent) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* ToString
* @return string
**/
func (s *Agent) ToString() string {
	result, err := s.ToJson()
	if err != nil {
		return ""
	}

	return result.ToString()
}

/**
* save
* @return error
**/
func (s *Agent) save() error {
	data, err := s.ToJson()
	if err != nil {
		return err
	}

	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.owner != nil && s.owner.store != nil {
		return s.owner.store.Set(s.ID, "agent", s)
	}

	return nil
}

/**
* up
* @param ia *Ia
**/
func (s *Agent) up(ia *Ia) {
	key := envar.GetStr("OPENAI_API_KEY", "")
	s.client = openai.NewClient(
		option.WithAPIKey(key),
	)
	s.owner = ia
}

/**
* conversations
* @param convID, prompt string
* @return (string, string, error)
**/
func (s *Agent) conversations(convID, prompt string) (et.Json, error) {
	if convID == "" {
		conv, _ := s.client.Conversations.New(s.ctx, conversations.ConversationNewParams{})
		convID = conv.ID
	}

	prompt = fmt.Sprintf(s.Context, prompt)
	result, err := s.client.Responses.New(s.ctx, responses.ResponseNewParams{
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
