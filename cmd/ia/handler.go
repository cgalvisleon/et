package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/ia"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

/**
* httpConversation: POST /conversation - the main "ask the model a question"
* endpoint. Body: {kb_id, statement}. Classifies statement against kb_id's current
* knowledge and returns the raw Verdict plus a human-readable message.
* @param w http.ResponseWriter, r *http.Request
**/
func httpConversation(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	kbId := body.Str("kb_id")
	statement := body.Str("statement")
	if statement == "" {
		response.HTTPError(w, r, http.StatusBadRequest, ia.MSG_STATEMENT_EMPTY)
		return
	}

	verdict, err := engine.Verify(kbId, statement)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := toJson(verdict)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	result.Set("message", conversationMessage(verdict))

	response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: result})
}

/**
* conversationMessage: turns a Verdict into a short, human-readable explanation for
* the /conversation endpoint.
* @param v ia.Verdict
* @return string
**/
func conversationMessage(v ia.Verdict) string {
	pct := int(v.Confidence * 100)

	switch {
	case v.IsTruth && v.ContradictsFactID != "":
		return fmt.Sprintf("Parece verdadero (%d%% de confianza), aunque contradice un hecho previo (%s).", pct, v.ContradictsFactID)
	case v.IsTruth:
		return fmt.Sprintf("Parece verdadero, con %d%% de confianza.", pct)
	case v.ContradictsFactID != "":
		return fmt.Sprintf("Parece una mentira (%d%% de confianza) y contradice un hecho ya conocido (%s).", pct, v.ContradictsFactID)
	default:
		return fmt.Sprintf("Parece una mentira, con %d%% de confianza.", pct)
	}
}

/**
* httpLearn: POST /learn - records a new truth. Body: {kb_id, statement, confidence,
* context}.
* @param w http.ResponseWriter, r *http.Request
**/
func httpLearn(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	kbId := body.Str("kb_id")
	statement := body.Str("statement")
	confidence := body.ValNum(1, "confidence")
	ctx := body.Json("context")

	fact, err := engine.Learn(kbId, statement, confidence, ctx)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := toJson(fact)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusCreated, et.Item{Ok: true, Result: result})
}

/**
* httpRevise: PUT /revise/{factId} - supersedes an existing fact with a new statement
* as more context changes what is known to be true. Body: {kb_id, statement,
* confidence, context}.
* @param w http.ResponseWriter, r *http.Request
**/
func httpRevise(w http.ResponseWriter, r *http.Request) {
	factId := r.PathValue("factId")

	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	kbId := body.Str("kb_id")
	statement := body.Str("statement")
	confidence := body.ValNum(1, "confidence")
	ctx := body.Json("context")

	fact, err := engine.Revise(kbId, factId, statement, confidence, ctx)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := toJson(fact)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: result})
}

/**
* httpVerify: POST /verify - classifies a statement as truth or lie against kb_id's
* current knowledge, without the /conversation endpoint's friendly message. Body:
* {kb_id, statement}.
* @param w http.ResponseWriter, r *http.Request
**/
func httpVerify(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	kbId := body.Str("kb_id")
	statement := body.Str("statement")
	if statement == "" {
		response.HTTPError(w, r, http.StatusBadRequest, ia.MSG_STATEMENT_EMPTY)
		return
	}

	verdict, err := engine.Verify(kbId, statement)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := toJson(verdict)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: result})
}

/**
* httpUnload: DELETE /unload/{kbId} - evicts a knowledge base from memory on demand.
* @param w http.ResponseWriter, r *http.Request
**/
func httpUnload(w http.ResponseWriter, r *http.Request) {
	kbId := r.PathValue("kbId")

	engine.Unload(kbId)

	response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: et.Json{
		"kb_id":    kbId,
		"unloaded": true,
	}})
}

/**
* httpIsLoaded: GET /loaded/{kbId} - reports whether a knowledge base currently sits
* in memory.
* @param w http.ResponseWriter, r *http.Request
**/
func httpIsLoaded(w http.ResponseWriter, r *http.Request) {
	kbId := r.PathValue("kbId")

	response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: et.Json{
		"kb_id":  kbId,
		"loaded": engine.IsLoaded(kbId),
	}})
}

/**
* toJson: marshals v (a Fact/Verdict/etc.) into an et.Json so it can be returned
* through et.Item.Result.
* @param v any
* @return et.Json, error
**/
func toJson(v any) (et.Json, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	result := et.Json{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}
