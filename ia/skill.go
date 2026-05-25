package ia

import (
	"context"
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/utility"
)

type Skill interface {
	Tag() string
	Name() string
	Description() string
	Execute(
		ctx context.Context,
		input et.Json,
	) (et.Json, error)
}

type ApiSkill struct {
	tag         string  `json:"-"`
	name        string  `json:"-"`
	description string  `json:"-"`
	Url         string  `json:"url"`
	Method      string  `json:"method"`
	Headers     et.Json `json:"headers"`
	Body        et.Json `json:"body"`
}

/**
* NewApiSkill
* @param string tag, name, description, url, method, headers, body
* @return *ApiSkill, error
**/
func NewApiSkill(tag, name, description, url, method string, headers et.Json, body et.Json) (*ApiSkill, error) {
	if utility.ValidStr(tag, 1, []string{}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "tag")
	}
	if utility.ValidStr(name, 1, []string{}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "name")
	}
	if utility.ValidStr(url, 1, []string{}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "url")
	}
	if utility.ValidStr(method, 1, []string{}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "method")
	}
	result := &ApiSkill{
		tag:         tag,
		name:        name,
		description: description,
		Url:         url,
		Method:      method,
		Headers:     headers,
		Body:        body,
	}
	return result, nil
}

/**
* Tag
* @return string
**/
func (s *ApiSkill) Tag() string {
	return s.tag
}

/**
* Name
* @return string
**/
func (s *ApiSkill) Name() string {
	return s.name
}

/**
* Description
* @return string
**/
func (s *ApiSkill) Description() string {
	return s.description
}

/**
* Execute
* @param context.Context ctx, et.Json input
* @return et.Json, error
**/
func (s *ApiSkill) Execute(ctx context.Context, input et.Json) (et.Json, error) {
	response, status := request.Fetch(s.Method, s.Url, s.Headers, s.Body)
	if !status.Ok {
		return nil, errors.New(status.Message)
	}

	result, err := response.ToJson()
	if err != nil {
		return nil, err
	}

	return result, nil
}
