package ia

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
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
	Tag     string          `json:"tag"`
	Context string          `json:"context"`
	Model   string          `json:"model"`
	client  openai.Client   `json:"-"`
	owner   *Agents         `json:"-"`
	ctx     context.Context `json:"-"`
	isDebug bool            `json:"-"`
}

/**
* newAgent
* @param owner *Agents, id, tag string
* @return *Agent
**/
func newAgent(owner *Agents, id, tag string) *Agent {
	if id == "" {
		id = reg.ULID()
	}

	return &Agent{
		ID:      id,
		Tag:     tag,
		Context: contextDefault,
		Model:   modelDefault,
		owner:   owner,
		ctx:     context.Background(),
		isDebug: owner.isDebug,
	}
}

/**
* Serialize
* @return ([]byte, error)
**/
func (s *Agent) Serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Agent) ToJson() et.Json {
	bt, err := s.Serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* ToString
* @return string
**/
func (s *Agent) ToString() string {
	return s.ToJson().ToString()
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

	return s.owner.saveAgent(s)
}

/**
* up
**/
func (s *Agent) up() {
	key := envar.GetStr("OPENAI_API_KEY", "")
	s.client = openai.NewClient(
		option.WithAPIKey(key),
	)
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
