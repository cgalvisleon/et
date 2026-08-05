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
* @param app, device, sessionID string
* @return string
**/
func GetKey(app, device, sessionID string) string {
	return fmt.Sprintf("%s:%s:%s", app, device, sessionID)
}

/**
* NewToken
* @param app, device, sessionID, name string, payload et.Json, duration time.Duration
* @return string, error
**/
func NewToken(app, device, sessionID, name string, payload et.Json, duration time.Duration) (string, error) {
	if !cache.IsLoad() {
		return "", errors.New(msg.MSG_CACHE_NOT_LOAD)
	}

	result, err := claim.NewToken(app, device, sessionID, name, payload, duration)
	if err != nil {
		return "", err
	}

	key := GetKey(app, device, sessionID)
	cache.SetWithDuration(key, result, duration)

	return result, nil
}

/**
* NewAuthentication
* @param app, device, sessionID, name string, duration time.Duration
* @return string, error
**/
func NewAuthentication(app, device, sessionID, name string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if sessionID == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "sessionID")
	}
	if name == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	return NewToken(app, device, sessionID, name, et.Json{}, duration)
}

/**
* NewAuthorization
* @param app, device, sessionID, name, tenantId, profileId string, duration time.Duration
* @return string, error
**/
func NewAuthorization(app, device, sessionID, name, tenantId, profileId string, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if sessionID == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "sessionID")
	}
	if name == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}
	if tenantId == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "tenantId")
	}
	if profileId == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "profileId")
	}
	payload := et.Json{
		"tenantId":  tenantId,
		"profileId": profileId,
	}

	return NewToken(app, device, sessionID, name, payload, duration)
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

	return NewToken(app, device, app, app, et.Json{}, duration)
}

/**
* NewEphemeralToken
* @param app, device, userId, username string, payload et.Json
* @return string, error
**/
func NewEphemeralToken(app, device, sessionID, name string, payload et.Json, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	if device == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "device")
	}
	if sessionID == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "username")
	}
	if name == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	maxDuration := 15 * time.Minute
	if duration > maxDuration {
		duration = maxDuration
	}

	return NewToken(app, device, sessionID, name, payload, duration)
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
* @param app, device, sessionID string
* @return error
**/
func DeleteToken(app, device, sessionID string) error {
	key := GetKey(app, device, sessionID)
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
	sessionID := parce.SessionID
	return DeleteToken(app, device, sessionID)
}

/**
* Validate
* @param ctx context.Context, token string
* @return *Claim, error
**/
func Validate(token string) (*claim.Claim, error) {
	clm, err := claim.ParceToken(token)
	if err != nil {
		return nil, err
	}

	app := clm.App
	device := clm.Device
	sessionID := clm.SessionID
	key := GetKey(app, device, sessionID)
	val, err := cache.Get(key, "")
	if err != nil {
		return nil, err
	}

	if val != token {
		cache.Delete(key)
		return nil, err
	}

	return clm, nil
}

/**
* SetToken
* @param app, device, sessionID, token string, duration time.Duration
* @return error
**/
func SetToken(app, device, sessionID, token string, duration time.Duration) error {
	key := GetKey(app, device, sessionID)
	if duration < 0 {
		cache.Delete(key)
		return errors.New(msg.MSG_TOKEN_EXPIRED)
	}

	cache.Set(key, token, duration)

	return nil
}

/**
* RenewToken
* @param token string, duration time.Duration
* @return string, error
**/
func RenewToken(token string, duration time.Duration) (string, error) {
	clm, err := Validate(token)
	if err != nil {
		return "", err
	}

	app := clm.App
	device := clm.Device
	sessionID := clm.SessionID
	key := GetKey(app, device, sessionID)
	result, err := NewToken(app, device, sessionID, clm.Name, clm.Payload, duration)
	if err != nil {
		return "", err
	}
	cache.Set(key, result, duration)
	return result, nil
}
