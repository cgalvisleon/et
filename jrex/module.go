package jrex

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

type Module struct {
	Tag      string  `json:"tag"`
	Metadata et.Json `json:"metadata"`
	Code     string  `json:"-"`
	jrex     *Jrex   `json:"-"`
}

/**
* NewModule: Creates a new module
* @param tag, userId string
* @return *Module
**/
func (s *Jrex) NewModule(tag, userId string) (*Module, error) {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return nil, errors.New(MSG_TAG_REQUIRED)
	}

	result := &Module{
		Tag:      tag,
		Metadata: et.Json{},
	}
	s.addModule(result, userId)
	return result, nil
}

/**
* SaveCode
* @param userId string
* @return error
**/
func (s *Module) saveCode(userId string) error {
	if s.jrex.store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	err := s.jrex.store.Set(storeCode, s.Tag, s.jrex.ID, s.Code)
	if err != nil {
		return err
	}

	return nil
}

/**
* getCode
* @return string, error
**/
func (s *Module) getCode() (string, error) {
	if s.jrex.store == nil {
		return "", errors.New(MSG_STORE_IS_NIL)
	}

	var result string
	exists, err := s.jrex.store.Get("code", s.Tag, &result)
	if err != nil {
		return "", err
	}

	if !exists {
		return "", errors.New(MSG_CODE_NOT_FOUND)
	}

	s.Code = result
	return s.Code, nil
}

/**
* up
* @param jrex *Jrex
* @return *Module
**/
func (s *Module) up(jrex *Jrex) *Module {
	s.jrex = jrex
	return s
}

/**
* SetMetadata
* @param metadata et.Json
**/
func (s *Module) SetMetadata(metadata et.Json) {
	s.Metadata = metadata
}
