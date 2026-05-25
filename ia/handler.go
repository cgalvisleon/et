package ia

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

/**
* HttpGetConversation
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpGetConversation(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	result, exists := s.getConversation(id)
	if !exists {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     true,
			Result: et.Json{"message": MSG_CONVERSATION_NOT_FOUND},
		})
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result.ToJson(),
	})
}

/**
* HttpGetAgent
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpGetAgent(w http.ResponseWriter, r *http.Request) {
	name := request.URLParam(r, "name").Str()
	result, exists := s.getAgent(name)
	if !exists {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     false,
			Result: et.Json{"message": MSG_AGENT_NOT_FOUND},
		})
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result.ToJson(),
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

	name := body.Str("name")
	description := body.Str("description")
	context := body.Str("context")
	model := body.Str("model")

	result, err := s.newAgent(name, description, context, model)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusCreated, et.Item{
		Ok:     true,
		Result: result.ToJson(),
	})
}

/**
* HttpRemoveAgent
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpRemoveAgent(w http.ResponseWriter, r *http.Request) {
	name := request.URLParam(r, "name").Str()
	s.removeAgent(name)

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": "agent removed"},
	})
}

/**
* HttpSetModelAgent
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpSetModelAgent(w http.ResponseWriter, r *http.Request) {
	agentName := request.URLParam(r, "agentName").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	model := body.Str("model")
	result, err := s.setModelAgent(agentName, model)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result.ToJson(),
	})
}

/**
* HttpSetContextAgent
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpSetContextAgent(w http.ResponseWriter, r *http.Request) {
	agentName := request.URLParam(r, "agentName").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	context := body.Str("context")
	result, err := s.setContextAgent(agentName, context)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result.ToJson(),
	})
}

/**
* HttpSetSkillAgent
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpSetSkillAgent(w http.ResponseWriter, r *http.Request) {
	// Skill is an interface with an Execute method that cannot be deserialized
	// from HTTP. Skills must be registered programmatically via setSkill.
	response.HTTPError(w, r, http.StatusNotImplemented, MSG_SKILL_HTTP_NOT_SUPPORTED)
}

/**
* HttpConversation
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Ia) HttpConversation(w http.ResponseWriter, r *http.Request) {
	agentName := request.URLParam(r, "agentName").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	convID := body.Str("convID")
	to := body.Str("to")
	prompt := body.Str("prompt")

	conversation, err := s.Conversation(r.Context(), agentName, convID, to, prompt)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: conversation.ToJson(),
	})
}
