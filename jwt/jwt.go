package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

/**
* GetTokenKey
* @param app, device, username string
* @return string
**/
func GetKey(app, device, username string) string {
	return fmt.Sprintf("%s:%s:%s", app, device, username)
}

/**
* New
* @param app, device, userId, username string, payload et.Json, duration time.Duration
* @return string, error
**/
func New(app, device, userId, username string, payload et.Json, duration time.Duration) (string, error) {
	if !cache.IsLoad() {
		return "", errors.New(msg.MSG_CACHE_NOT_LOAD)
	}

	result, err := claim.NewToken(app, device, userId, username, payload, duration)
	if err != nil {
		return "", err
	}

	key := GetKey(app, device, username)
	cache.SetDuration(key, result, duration)

	return result, nil
}

/**
* NewAuthentication
* @param app, device, userId, username string, duration time.Duration
* @return string, error
**/
func NewAuthentication(app, device, userId, username string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if username == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "username")
	}

	return New(app, device, userId, username, et.Json{}, duration)
}

/**
* NewAuthorization
* @param app, device, userId, username, tenantId, profileTp string, duration time.Duration
* @return string, error
**/
func NewAuthorization(app, device, userId, username, tenantId, profileTp string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if username == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "username")
	}
	if tenantId == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tenantId")
	}
	if profileTp == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "profileTp")
	}

	return New(app, device, userId, username, et.Json{
		"tenant_id":  tenantId,
		"profile_tp": profileTp,
	}, duration)
}

/**
* NewAppToken
* @param app, device string, duration time.Duration
* @return string, error
**/
func NewAppToken(app, device string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}

	return New(app, device, app, app, et.Json{}, duration)
}

/**
* NewEphemeralToken
* @param app, device, userId, username string, payload et.Json, duration time.Duration
* @return string, error
**/
func NewEphemeralToken(app, device, userId, username string, payload et.Json, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if username == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "username")
	}
	if duration <= 0 {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "duration")
	}

	return New(app, device, userId, username, payload, duration)
}

/**
* GetToken
* @param key string
* @return string, error
**/
func GetToken(key string) (string, error) {
	return cache.Get(key, "")
}

/**
* DeleteToken
* @param app, device, username string
* @return error
**/
func DeleteToken(app, device, username string) error {
	key := GetKey(app, device, username)
	_, err := cache.Delete(key)
	if err != nil {
		return err
	}

	return nil
}

/**
* DeleteTokeByToken
* @param token string
* @return error
**/
func DeleteTokeByToken(token string) error {
	parce, err := claim.ParceToken(token)
	if err != nil {
		return err
	}

	app := parce.App
	device := parce.Device
	username := parce.Username
	return DeleteToken(app, device, username)
}

/**
* Validate
* @param token string
* @return *Claim, error
**/
func Validate(token string) (*claim.Claim, error) {
	result, err := claim.ParceToken(token)
	if err != nil {
		return nil, err
	}

	app := result.App
	device := result.Device
	username := result.Username

	key := GetKey(app, device, username)
	val, err := cache.Get(key, "")
	if err != nil {
		return nil, err
	}

	if val != token {
		cache.Delete(key)
		return nil, err
	}

	return result, nil
}

/**
* SetToken
* @param app, device, username, token string, duration time.Duration
* @return error
**/
func SetToken(app, device, username, token string, duration time.Duration) error {
	key := GetKey(app, device, username)
	if duration < 0 {
		cache.Delete(key)
		return errors.New(msg.MSG_TOKEN_EXPIRED)
	}

	cache.Set(key, token, duration)

	return nil
}
