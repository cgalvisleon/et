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
* @param app, device, username string, payload et.Json, duration time.Duration
* @return string, error
**/
func New(app, device, username string, payload et.Json, duration time.Duration) (string, error) {
	if !cache.IsLoad() {
		return "", errors.New(msg.MSG_CACHE_NOT_LOAD)
	}

	result, err := claim.NewToken(app, device, username, payload, duration)
	if err != nil {
		return "", err
	}

	key := GetKey(app, device, username)
	cache.SetDuration(key, result, duration)

	return result, nil
}

/**
* NewAuthentication
* @param app, device, username string, duration time.Duration
* @return string, error
**/
func NewAuthentication(app, device, username string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if username == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "username")
	}

	return New(app, device, username, et.Json{}, duration)
}

/**
* NewAuthorization
* @param app, device, username, tenantId, profileTp string, duration time.Duration
* @return string, error
**/
func NewAuthorization(app, device, username, tenantId, profileTp string, duration time.Duration) (string, error) {
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

	return New(app, device, username, et.Json{
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

	return New(app, device, app, et.Json{}, duration)
}

/**
* NewAppTagToken
* @param app, device, tag string, duration time.Duration
* @return string, error
**/
func NewAppTagToken(app, device, tag string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if tag == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tag")
	}

	return New(app, device, app, et.Json{
		"tag": tag,
	}, duration)
}

/**
* NewEphemeralToken
* @param app, device, username string, duration time.Duration
* @return string, error
**/
func NewEphemeralToken(app, device, username string, payload et.Json, duration time.Duration) (string, error) {
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

	return New(app, device, username, payload, duration)
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
	claim, err := claim.ParceToken(token)
	if err != nil {
		return err
	}

	app := claim.Payload.Str("app")
	device := claim.Payload.Str("device")
	username := claim.Payload.Str("username")
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

	payload := result.Payload
	app := payload.Str("app")
	device := payload.Str("device")
	username := payload.Str("username")

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
