package ia

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

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
	name := request.URLParam(r, "name").Str()
	userId := request.UserId(r)
	err := s.removeAgent(name, userId)
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
		Result: et.Json{"message": "agent updated"},
	})
}

/**
* HttpDelete
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpDelete(w http.ResponseWriter, r *http.Request) {
	err := s.delete()
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
* HttpConversation
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpConversation(w http.ResponseWriter, r *http.Request) {
	agentName := request.URLParam(r, "name").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	to := body.Str("to")
	prompt := body.Str("prompt")
	userId := request.UserId(r)

	conversation, err := s.Conversation(r.Context(), agentName, to, prompt, userId)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: conversation.ToJson(),
	})
}
