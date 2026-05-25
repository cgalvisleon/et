package ia

import (
	"context"

	"github.com/cgalvisleon/et/et"
)

type Skill interface {
	Tag() string
	Name() string
	Description() string
	Execute(
		ctx context.Context,
		input et.Json,
	) (*SkillResult, error)
}

type SkillResult struct {
	Success bool
	Data    any
	Error   string
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
* @return *ApiSkill
**/
func NewApiSkill(tag, name, description, url, method string, headers et.Json, body et.Json) *ApiSkill {
	return &ApiSkill{
		tag:         tag,
		name:        name,
		description: description,
		Url:         url,
		Method:      method,
		Headers:     headers,
		Body:        body,
	}
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
* @return *SkillResult, error
**/
func (s *ApiSkill) Execute(ctx context.Context, input et.Json) (*SkillResult, error) {
	return nil, nil
}
