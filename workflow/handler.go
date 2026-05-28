package workflow

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

/**
* HttpGetFlow
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpGetFlow(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	result, exists := s.getFlow(id)
	if !exists {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     true,
			Result: et.Json{"message": MSG_FLOW_NOT_FOUND},
		})
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result.ToJson(),
	})
}

/**
* HttpNewFlow
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpNewFlow(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	tag := body.Str("tag")
	version := body.Str("version")
	name := body.Str("name")
	description := body.Str("description")
	username := body.Str("username")

	flow, err := s.NewFlow(tag, version, name, description, username)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusCreated, et.Item{
		Ok:     true,
		Result: flow.ToJson(),
	})
}

/**
* HttpDeleteFlow
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpDeleteFlow(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	err := s.DeleteFlow(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": fmt.Sprintf(MSG_WORKFLOW_DELETE, id)},
	})
}

/**
* HttpGetInstance
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpGetInstance(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()

	result, err := s.GetInstance(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if result == nil {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     false,
			Result: et.Json{"message": MSG_INSTANCE_NOT_FOUND},
		})
		return
	}

	item := result.ToJson()
	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: item,
	})
}

/**
* HttpDeleteInstance
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpDeleteInstance(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	err := s.DeleteInstance(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": fmt.Sprintf(MSG_INSTANCE_DELETE, id)},
	})
}

/**
* HttpRunInstance
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpRunInstance(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	tag := body.Str("tag")
	id := body.Str("id")
	ownerId := body.Str("ownerId")
	step := body.Int("step")
	ctx := body.Json("ctx")
	tags := body.Json("tags")
	username := body.Str("username")

	result, err := s.RunInstance(tag, id, ownerId, step, ctx, tags, username)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result,
	})
}

/**
* HttpResetInstance
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpResetInstance(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	username := body.Str("username")

	result, err := s.ResetInstance(id, username)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result,
	})
}

/**
* HttpRollbackInstance
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpRollbackInstance(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	username := body.Str("username")

	result, err := s.RollbackInstance(id, username)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result,
	})
}

/**
* HttpStopInstance
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpStopInstance(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	username := body.Str("username")

	result, err := s.StopInstance(id, username)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result,
	})
}

/**
* HttpAddStepFromSteper
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpAddStepFromSteper(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	flowTag := body.Str("flow_tag")
	tag := body.Str("tag")
	index := body.Int("index")

	flow, ok := s.AddStepFromSteper(flowTag, tag, index)
	if !ok {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     false,
			Result: et.Json{"message": MSG_FLOW_NOT_FOUND},
		})
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: flow.ToJson(),
	})
}

/**
* HttpRemoveStepFromSteper
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpRemoveStepFromSteper(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	tag := request.URLParam(r, "tag").Str()
	index := request.URLParam(r, "index").Int()

	flow, ok := s.RemoveStepFromSteper(id, tag, index)
	if !ok {
		response.ITEM(w, r, http.StatusNotFound, et.Item{
			Ok:     false,
			Result: et.Json{"message": MSG_FLOW_NOT_FOUND},
		})
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: flow.ToJson(),
	})
}

/**
* HttpMoveStepFromSteper
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpMoveStepFromSteper(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	tag := body.Str("tag")
	index := body.Int("index")
	to := body.Int("to")

	result, err := s.MoveStepFromSteper(id, tag, index, to)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result,
	})
}

/**
* HttpNewSteper
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpNewSteper(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	flowTag := body.Str("flow_tag")
	tag := body.Str("tag")
	name := body.Str("name")
	description := body.Str("description")

	result, err := s.NewSteper(flowTag, tag, name, description)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusCreated, et.Item{
		Ok:     true,
		Result: result,
	})
}

/**
* HttpSetSteper
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpSetSteper(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	tag := body.Str("tag")
	name := body.Str("name")
	description := body.Str("description")

	result, err := s.SetSteper(id, tag, name, description)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result,
	})
}

/**
* HttpDeleteSteper
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpDeleteSteper(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	tag := request.URLParam(r, "tag").Str()

	err := s.DeleteSteper(id, tag)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": "ok"},
	})
}

/**
* HttpNewStep
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpNewStep(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	flowTag := body.Str("flow_tag")
	name := body.Str("name")
	description := body.Str("description")
	definition := body.Str("definition")
	undo := body.Str("undo")
	stop := body.Bool("stop")

	result, err := s.NewStep(flowTag, name, description, definition, undo, stop)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusCreated, et.Item{
		Ok:     true,
		Result: result,
	})
}

/**
* HttpSetStep
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpSetStep(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	index := body.Int("index")
	name := body.Str("name")
	description := body.Str("description")
	definition := body.Str("definition")
	undo := body.Str("undo")
	stop := body.Bool("stop")

	result, err := s.SetStep(id, index, name, description, definition, undo, stop)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result,
	})
}

/**
* HttpDeleteStep
* @params w http.ResponseWriter, r *http.Request
**/
func (s *WorkFlow) HttpDeleteStep(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	index := request.URLParam(r, "index").Int()

	_, err := s.DeleteStep(id, index)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": "ok"},
	})
}
