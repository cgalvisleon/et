package claim

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/golang-jwt/jwt/v4"
)

type Claim struct {
	jwt.StandardClaims
	Salt     string        `json:"salt"`
	Duration time.Duration `json:"duration"`
	App      string        `json:"app"`
	Device   string        `json:"device"`
	Username string        `json:"username"`
	Payload  et.Json       `json:"payload"`
}

/**
* ToJson
* @return et.Json
**/
func (s *Claim) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	result := et.Json{}
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Unix(s.ExpiresAt, 0).Format("2006-01-02 03:04:05 PM")
	result.Set("expiresAt", expiresAt)

	return result, nil
}

/**
* SetPayload
* @param payload et.Json
**/
func (s *Claim) SetPayload(payload et.Json) {
	s.Payload = payload
}

/**
* NewClaim
* @param duration time.Duration
* @return *Claim
**/
func NewClaim(duration time.Duration) *Claim {
	result := &Claim{}
	result.Salt = utility.GetOTP(6)
	result.Duration = duration
	if result.Duration != 0 {
		result.ExpiresAt = timezone.Add(result.Duration).Unix()
	}

	return result
}

/**
* GenToken
* @param c *Claim, secret string
* @return string, error
**/
func GenToken(c *Claim, secret string) (string, error) {
	_jwt := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	result, err := _jwt.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return result, nil
}

/**
* GetTokenKey
* @param app, device, username string
* @return string
**/
func GetTokenKey(app, device, username string) string {
	return fmt.Sprintf("%s:%s:%s", app, device, username)
}

/**
* NewToken
* @param app, device, username string, payload et.Json, duration time.Duration
* @return string, string, error
**/
func NewToken(app, device, username string, payload et.Json, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	c := NewClaim(duration)
	c.App = app
	c.Device = device
	c.Username = username
	c.Payload = payload
	secret := envar.GetStr("SECRET", "1977")
	result, err := GenToken(c, secret)
	if err != nil {
		return "", err
	}

	key := GetTokenKey(app, device, username)
	cache.SetDuration(key, result, c.Duration)

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

	return NewToken(app, device, username, et.Json{}, duration)
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

	return NewToken(app, device, username, et.Json{
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

	return NewToken(app, device, app, et.Json{}, duration)
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

	return NewToken(app, device, app, et.Json{
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

	return NewToken(app, device, username, payload, duration)
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
	key := GetTokenKey(app, device, username)
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
	claim, err := ParceToken(token)
	if err != nil {
		return err
	}

	app := claim.Payload.Str("app")
	device := claim.Payload.Str("device")
	username := claim.Payload.Str("username")
	return DeleteToken(app, device, username)
}

/**
* ParceToken
* @param token string
* @return *Claim, error
**/
func ParceToken(token string) (*Claim, error) {
	secret := envar.GetStr("SECRET", "1977")
	jToken, err := jwt.Parse(token, func(*jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if !jToken.Valid {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID)
	}

	claim, ok := jToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_REQUIRED_INVALID)
	}

	payload, ok := claim["payload"].(et.Json)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "payload")
	}

	second, ok := claim["duration"].(float64)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "duration")
	}

	duration := time.Duration(second)
	result := &Claim{
		Duration: duration,
		Payload:  payload,
	}
	if result.Duration != 0 {
		exp, ok := claim["exp"].(float64)
		if ok {
			result.ExpiresAt = int64(exp)
		}
	}

	return result, nil
}

/**
* ValidToken
* @param token string
* @return *Claim, error
**/
func ValidToken(token string) (*Claim, error) {
	result, err := ParceToken(token)
	if err != nil {
		return nil, err
	}

	payload := result.Payload
	app := payload.Str("app")
	device := payload.Str("device")
	username := payload.Str("username")

	key := GetTokenKey(app, device, username)
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
	key := GetTokenKey(app, device, username)
	if duration < 0 {
		cache.Delete(key)
		return fmt.Errorf(msg.MSG_TOKEN_EXPIRED)
	}

	cache.Set(key, token, duration)

	return nil
}
