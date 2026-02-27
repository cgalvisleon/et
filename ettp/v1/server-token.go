package ettp

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jwt"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
)

/**
* developToken
* @return string
**/
func developToken() string {
	production := envar.GetBool("PRODUCTION", true)
	if production {
		return ""
	}

	device := "develop"
	duration := 1 * time.Hour
	token, err := claim.NewToken(device, device, device, et.Json{}, duration)
	if err != nil {
		logs.Alert(err)
		return ""
	}

	_, err = jwt.Validate(token)
	if err != nil {
		logs.Alertf("developToken:%s", err.Error())
		return ""
	}

	return token
}

/**
* GetTokenByKey
* @param key string
* @return error
**/
func (s *Server) GetTokenByKey(key string) (et.Item, error) {
	if !utility.ValidStr(key, 0, []string{}) {
		return et.Item{}, logs.Alertf(msg.MSG_ATRIB_REQUIRED, "key")
	}

	result, err := cache.Get(key, "")
	if err != nil {
		return et.Item{}, err
	}

	if result == "" {
		return et.Item{}, errors.New(msg.MSG_RECORD_NOT_FOUND)
	}

	valid := MSG_TOKEN_VALID
	_, err = jwt.Validate(result)
	if err != nil {
		valid = err.Error()
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"key":   key,
			"value": result,
			"valid": valid,
		},
	}, nil
}

/**
* handlerValidToken
* @param key string
* @return error
**/
func (s *Server) HandlerValidToken(key string) (et.Item, error) {
	if !utility.ValidStr(key, 0, []string{}) {
		return et.Item{}, logs.Alertf(msg.MSG_ATRIB_REQUIRED, "key")
	}

	result, err := cache.Get(key, "")
	if err != nil {
		return et.Item{}, err
	}

	if result == "" {
		return et.Item{}, errors.New(msg.MSG_RECORD_NOT_FOUND)
	}

	_, err = jwt.Validate(result)
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"key":   key,
			"value": result,
		},
	}, nil
}

/**
* DeleteTokenByKey
* @param id string
* @return error
**/
func (s *Server) DeleteTokenByKey(key string) error {
	if !utility.ValidStr(key, 0, []string{}) {
		return logs.Alertf(msg.MSG_ATRIB_REQUIRED, "key")
	}

	_, err := cache.Delete(key)
	if err != nil {
		return err
	}

	return nil
}
