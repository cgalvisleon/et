package ia

import (
	"encoding/json"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const contextDefault = `Eres un asistente que SOLO puede responder con base en el CONTEXTO dado.

Reglas obligatorias:
1. Usa únicamente información del CONTEXTO.
2. No completes con conocimiento externo.
3. No hagas suposiciones.
4. Si la respuesta no está explícitamente en el contexto, responde exactamente:
"No tengo suficiente información para responder a tu pregunta."
`

type Agent struct {
	ID      string        `json:"id"`
	Tag     string        `json:"tag"`
	Context string        `json:"context"`
	Model   string        `json:"model"`
	client  openai.Client `json:"-"`
	owner   *Agents       `json:"-"`
	isDebug bool          `json:"-"`
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
* Save
* @return error
**/
func (s *Agent) Save() error {
	data := s.ToJson()
	event.Publish(EVENT_AGENT_STATUS, data)

	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.owner != nil && s.owner.setInstance != nil {
		return s.owner.setInstance(s.ID, s.Tag, s)
	}

	return nil
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
