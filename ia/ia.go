package ia

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/openai/openai-go/v3"
)

var (
	packageName = "ia"
	ia          *Ia
)

type Ia struct {
	ID                string                   `json:"id"`
	Tag               string                   `json:"tag"`
	Agents            map[string]*Agent        `json:"agents"`
	Participants      map[string]*Participant  `json:"participants"`
	Conversations     map[string]*Conversation `json:"-"`
	db                *jsql.DB                 `json:"-"`
	sender            Sender                   `json:"-"`
	muAgents          sync.RWMutex             `json:"-"`
	muParticipants    sync.RWMutex             `json:"-"`
	muConversations   sync.RWMutex             `json:"-"`
	key               string                   `json:"-"`
	store             jsql.Store               `json:"-"`
	participantStore  jsql.Store               `json:"-"`
	conversationStore jsql.Store               `json:"-"`
	isDebug           bool                     `json:"-"`
}

/**
* New
* @param tag string, db *jsql.DB
* @return (*Ia, error)
**/
func New(tag string, db *jsql.DB) (*Ia, error) {
	err := event.Load()
	if err != nil {
		return nil, err
	}

	key := envar.GetStr("OPENAI_API_KEY", "")
	result := &Ia{
		ID:              "ia:" + tag,
		Tag:             tag,
		Agents:          make(map[string]*Agent, 0),
		Participants:    make(map[string]*Participant, 0),
		Conversations:   make(map[string]*Conversation, 0),
		muAgents:        sync.RWMutex{},
		muParticipants:  sync.RWMutex{},
		muConversations: sync.RWMutex{},
		isDebug:         envar.GetBool("DEBUG", false),
		key:             key,
		db:              db,
	}
	err = result.up()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* New
* @param db *jsql.DB
* @return error
**/
func Load(tag string, db *jsql.DB) error {
	if ia != nil {
		return nil
	}

	var err error
	ia, err = New(tag, db)
	if err != nil {
		return err
	}

	return nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Ia) ToJson() et.Json {
	return et.Json{
		"id":            s.ID,
		"tag":           s.Tag,
		"agents":        s.Agents,
		"conversations": s.Conversations,
	}
}

/**
* save
* @return error
**/
func (s *Ia) save() error {
	data := s.ToJson()
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	if s.store != nil {
		err := s.store.Set(s.ID, packageName, "", s)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_IA_SET, data)
	return nil
}

/**
* delete
* @return error
**/
func (s *Ia) delete() error {
	if s.store != nil {
		err := s.store.Delete(s.ID)
		if err != nil {
			return err
		}
	}

	event.Publish(EVENT_IA_DELETE, et.Json{
		"id": s.ID,
	})
	return nil
}

/**
* up
**/
func (s *Ia) up() error {
	var err error
	if s.store == nil {
		s.store, err = jsql.DefineInstance(s.db, "ia", "store", jsql.KindJson)
		if err != nil {
			return err
		}
	}

	if s.participantStore == nil {
		s.participantStore, err = jsql.DefineInstance(s.db, "ia", "participant", jsql.KindJson)
		if err != nil {
			return err
		}
	}

	if s.conversationStore == nil {
		s.conversationStore, err = jsql.DefineInstance(s.db, "ia", "conversation", jsql.KindJson)
		if err != nil {
			return err
		}
	}

	err = s.loadAgents()
	if err != nil {
		return err
	}

	return nil
}

/**
* loadAgents
* @return error
**/
func (s *Ia) loadAgents() error {
	var res *Ia
	exists, err := s.store.Get(s.ID, &res)
	if err != nil {
		return err
	}

	if exists && res != nil {
		for k, v := range res.Agents {
			v.up(s)
			s.Agents[k] = v
		}
	}

	return nil
}

/**
* addAgent
* @param agent *Agent
**/
func (s *Ia) addAgent(agent *Agent) {
	s.muAgents.Lock()
	defer s.muAgents.Unlock()

	s.Agents[agent.ID] = agent
}

/**
* getAgent
* @param tag string
* @return (*Agent, bool)
**/
func (s *Ia) getAgent(tag string) (*Agent, bool) {
	id := agendId(tag)
	s.muAgents.RLock()
	result, exists := s.Agents[id]
	s.muAgents.RUnlock()
	if exists {
		return result, true
	}

	if s.store != nil {
		exists, err := s.store.Get(id, &result)
		if err != nil {
			return nil, false
		}

		if exists {
			result.up(s)
			s.addAgent(result)
			return result, true
		}
	}

	return nil, false
}

/**
* removeAgent
* @param tag string
* @return error
**/
func (s *Ia) removeAgent(tag string) error {
	id := agendId(tag)
	s.muAgents.Lock()
	defer s.muAgents.Unlock()

	delete(s.Agents, id)
	return s.save()
}

/**
* newAgent
* @param tag, name, description, context, model string
* @return (*Agent, error)
**/
func (s *Ia) newAgent(tag, name, description, context, model string) (*Agent, error) {
	_, exists := s.getAgent(name)
	if exists {
		return nil, fmt.Errorf(MSG_AGENT_ALREADY_EXISTS, name)
	}

	result := newAgent(s, tag, name, description, context, model)
	s.addAgent(result)
	return result, s.save()
}

/**
* setModelAgent
* @param name string, model string
* @return (*Agent, error)
**/
func (s *Ia) SetModelAgent(name string, model string) (*Agent, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}
	if !utility.ValidStr(model, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "model")
	}

	result, exists := s.getAgent(name)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, name)
	}
	result.setModel(model)
	return result, s.save()
}

/**
* setContextAgent
* @param name string, context string
* @return (*Agent, error)
**/
func (s *Ia) SetContextAgent(name string, context string) (*Agent, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}
	if !utility.ValidStr(context, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "context")
	}

	result, exists := s.getAgent(name)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, name)
	}
	result.setContext(context)
	return result, s.save()
}

/**
* SetSkillAgent
* @param name string, skill Skill
* @return (*Agent, error)
**/
func (s *Ia) SetSkillAgent(name string, skill Skill) (*Agent, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}
	if !utility.ValidStr(skill.Tag(), 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "skill")
	}

	result, exists := s.getAgent(name)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, name)
	}
	result.addSkill(skill)
	return result, s.save()
}

/**
* loadParticipant
* @param to string, dest any
* @return (*Participant, error)
**/
func (s *Ia) loadParticipant(to string, dest *Participant) (bool, error) {
	items, err := s.participantStore.
		Query(et.Json{
			"where": et.Json{
				"to": et.Json{
					"eq": to,
				},
			},
			"limit": 1,
		})
	if err != nil {
		return false, err
	}

	if !items.Ok {
		return false, fmt.Errorf(MSG_PARTICIPANT_NOT_FOUND)
	}

	item, err := items.First()
	if err != nil {
		return false, err
	}

	bt := []byte(item.ToString())
	err = json.Unmarshal(bt, dest)
	if err != nil {
		return false, err
	}

	return true, nil
}

/**
* getParticipant
* @param userId, to, name string, role Role
* @return (*Participant, error)
**/
func (s *Ia) getParticipant(to, name string, role Role) (*Participant, error) {
	s.muParticipants.Lock()
	result, exists := s.Participants[to]
	s.muParticipants.Unlock()
	if exists {
		return result, nil
	}

	exists, err := s.loadParticipant(to, result)
	if err != nil {
		return nil, err
	}
	if !exists {
		result, err = newParticipant(s, "", to, name, role)
		if err != nil {
			return nil, err
		}
	}

	s.muParticipants.Lock()
	s.Participants[to] = result
	s.muParticipants.Unlock()

	return result, s.save()
}

/**
* loadConversation
* @param to string, dest *Conversation
* @return (bool, error)
**/
func (s *Ia) loadConversation(to string, dest *Conversation) (bool, error) {
	items, err := s.participantStore.
		Query(et.Json{
			"where": et.Json{
				"to": et.Json{
					"eq": to,
				},
			},
			"limit": 1,
		})
	if err != nil {
		return false, err
	}

	if !items.Ok {
		return false, fmt.Errorf(MSG_PARTICIPANT_NOT_FOUND)
	}

	item, err := items.First()
	if err != nil {
		return false, err
	}

	bt := []byte(item.ToString())
	err = json.Unmarshal(bt, dest)
	if err != nil {
		return false, err
	}

	return true, nil
}

/**
* getConversation
* @param to *Participant
* @return (*Conversation, error)
**/
func (s *Ia) getConversation(to *Participant) (*Conversation, error) {
	s.muConversations.RLock()
	result, exists := s.Conversations[to.To]
	s.muConversations.RUnlock()
	if !exists {
		return result, nil
	}

	exists, err := s.loadConversation(to.To, result)
	if err != nil {
		return nil, err
	}
	if !exists {
		result, err = newConversation(to, to.Name, Direct)
		if err != nil {
			return nil, err
		}
	}

	s.muConversations.Lock()
	s.Conversations[to.To] = result
	s.muConversations.Unlock()

	return result, nil
}

/**
* Embed - Genera un embedding
* @param ctx context.Context, agentName string, text string
* @return ([]float64, error)
**/
func (s *Ia) Embed(ctx context.Context, agentName string, text string) ([]float64, error) {
	if !utility.ValidStr(agentName, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "agentName")
	}
	if !utility.ValidStr(text, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "text")
	}

	result, exists := s.getAgent(agentName)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, agentName)
	}

	client := result.client
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModelTextEmbedding3Small,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
	})
	if err != nil {
		return nil, err
	}

	return resp.Data[0].Embedding, nil
}

/**
* Conversation
* @param ctx context.Context, agentName string, convID string, to string, prompt string
* @return (*Conversation, error)
**/
func (s *Ia) Conversation(ctx context.Context, agentName, to, prompt string) (*Conversation, error) {
	if !utility.ValidStr(agentName, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "agentName")
	}
	if !utility.ValidStr(prompt, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "prompt")
	}

	agent, exists := s.getAgent(agentName)
	if !exists {
		return nil, fmt.Errorf(MSG_AGENT_NOT_FOUND, agentName)
	}

	participant, err := s.getParticipant(to, to, Member)
	if err != nil {
		return nil, err
	}

	conversation, err := s.getConversation(participant)
	if err != nil {
		return nil, err
	}

	response, err := agent.conversation(ctx, conversation, prompt)
	if err != nil {
		return nil, err
	}

	_, err = conversation.SendTextMessage(response.Text)
	if err != nil {
		return nil, err
	}

	return conversation, nil
}
