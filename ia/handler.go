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
