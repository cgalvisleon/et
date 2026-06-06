package ia

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/router"
)

func (s *Ia) LoadRouter(r router.Router) {
	r.Protect(router.Get, "/agents/{tag}", s.HttpGetAgent)
	r.Protect(router.Post, "/agents", s.HttpNewAgent)
	r.Protect(router.Delete, "/agents/{tag}", s.HttpDeleteAgent)
	r.Protect(router.Put, "/agents/{tag}", s.HttpSetAgent)
	r.Protect(router.Post, "/conversation", s.HttpConversation)
	r.Protect(router.Delete, "/conversations/{to}", s.HttpDeleteConversation)
	r.Protect(router.Get, "/participants/{to}", s.HttpGetParticipant)
	r.Protect(router.Post, "/participants", s.HttpNewParticipant)
	r.Protect(router.Delete, "/participants/{to}", s.HttpDeleteParticipant)
	r.Protect(router.Put, "/participants/{to}", s.HttpSetParticipant)
}

/**
* HttpGetAgent
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpGetAgent(w http.ResponseWriter, r *http.Request) {
	tag := request.URLParam(r, "tag").Str()
	agent, exists := s.getAgent(tag)
	if !exists {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     false,
			Result: et.Json{"message": MSG_AGENT_NOT_FOUND},
		})
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: agent.ToJson(),
	})
}

/**
* HttpNewAgent
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpNewAgent(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	tag := body.Str("tag")
	name := body.Str("name")
	description := body.Str("description")
	context := body.Str("context")
	model := body.Str("model")
	userId := request.UserId(r)

	agent, err := s.newAgent(tag, name, description, context, model, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusCreated, et.Item{
		Ok:     true,
		Result: agent.ToJson(),
	})
}

/**
* HttpDeleteAgent
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpDeleteAgent(w http.ResponseWriter, r *http.Request) {
	tag := request.URLParam(r, "tag").Str()
	userId := request.UserId(r)
	err := s.deleteAgent(tag, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": "agent removed"},
	})
}

/**
* HttpSetAgent
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpSetAgent(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	tag := request.URLParam(r, "tag").Str()
	userId := request.UserId(r)

	agent, exists := s.getAgent(tag)
	if !exists {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     false,
			Result: et.Json{"message": MSG_AGENT_NOT_FOUND},
		})
		return
	}

	model := body.Str("model")
	if model != "" {
		agent.setModel(model)
	}

	context := body.Str("context")
	if context != "" {
		agent.setContext(context)
	}

	skillDef := body.Json("skill")
	if skillDef != nil {
		skill, err := NewApiSkill(
			skillDef.Str("tag"),
			skillDef.Str("name"),
			skillDef.Str("description"),
			skillDef.Str("url"),
			skillDef.Str("method"),
			skillDef.Json("headers"),
			skillDef.Json("body"),
		)
		if err != nil {
			response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		agent.addSkill(skill)
	}

	if agent.isChanged {
		err = agent.save(userId)
		if err != nil {
			response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": MSG_AGENT_UPDATED},
	})
}

/**
* HttpConversation
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpConversation(w http.ResponseWriter, r *http.Request) {
	tagAgent := request.URLParam(r, "tag").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	to := body.Str("to")
	prompt := body.Str("prompt")
	userId := request.UserId(r)

	ctx := r.Context()
	conversation, err := s.Conversation(ctx, tagAgent, to, prompt, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: conversation.ToJson(),
	})
}

/**
* HttpDeleteConversation
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpDeleteConversation(w http.ResponseWriter, r *http.Request) {
	to := request.URLParam(r, "to").Str()
	userId := request.UserId(r)
	err := s.deleteConversation(to, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": "ia deleted"},
	})
}

/**
* HttpGetParticipant
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpGetParticipant(w http.ResponseWriter, r *http.Request) {
	to := request.URLParam(r, "to").Str()
	userId := request.UserId(r)
	participant, err := s.getParticipant(to, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: participant.ToJson(),
	})
}

/**
* HttpNewParticipant
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpNewParticipant(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	to := body.Str("to")
	id := body.Str("user_id")
	name := body.Str("name")
	userId := request.UserId(r)

	participant, err := s.newParticipant(to, id, name, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusCreated, et.Item{
		Ok: true,
		Result: et.Json{
			"id":      participant.ID,
			"message": MSG_PARTICIPANT_CREATED,
		},
	})
}

/**
* HttpDeleteParticipant
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpDeleteParticipant(w http.ResponseWriter, r *http.Request) {
	to := request.URLParam(r, "to").Str()
	userId := request.UserId(r)
	err := s.deleteParticipant(to, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": MSG_PARTICIPANT_DELETED},
	})
}

/**
* HttpSetParticipant
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpSetParticipant(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	to := request.URLParam(r, "to").Str()
	userId := request.UserId(r)

	participant, err := s.getParticipant(to, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	id := body.Str("user_id")
	if id != "" {
		participant.SetUserId(id)
	}

	name := body.Str("name")
	if name != "" {
		participant.SetName(name)
	}

	if participant.isChanged {
		err = participant.save(userId)
		if err != nil {
			response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": MSG_PARTICIPANT_UPDATED},
	})
}
