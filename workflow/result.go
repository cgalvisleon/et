package workflow

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

type Result struct {
	Step    int     `json:"step"`
	Ctx     et.Json `json:"ctx"`
	Attempt int     `json:"attempt"`
	Result  et.Json `json:"result"`
	Error   string  `json:"error"`
}

/**
* Serialize
* @return string
**/
func (s *Result) Serialize() (string, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return "", err
	}

	return string(bt), nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Result) ToJson() et.Json {
	return et.Json{
		"step":    s.Step,
		"ctx":     s.Ctx,
		"attempt": s.Attempt,
		"result":  s.Result,
		"error":   s.Error,
	}
}

type resultFn struct {
	Result et.Json `json:"result"`
	Error  error   `json:"error"`
}

/**
* ToJson
* @return et.Json
**/
func (s *resultFn) ToJson() et.Json {
	return et.Json{
		"result": s.Result,
		"error":  s.Error,
	}
}

/**
* Serialize
* @return string
**/
func (s *resultFn) Serialize() (string, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return "", err
	}

	return string(bt), nil
}

/**
* loadResultFn
* @param s string
* @return *resultFn
**/
func loadResultFn(s string) (*resultFn, error) {
	var result resultFn
	err := json.Unmarshal([]byte(s), &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
