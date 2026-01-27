package claim

import (
	"encoding/json"
	"fmt"
	"time"

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
* NewToken
* @param app, device, username string, payload et.Json, duration time.Duration
* @return string, error
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

	return result, nil
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

	app, ok := claim["app"].(string)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "app")
	}

	device, ok := claim["device"].(string)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "device")
	}

	username, ok := claim["username"].(string)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "username")
	}

	payloadItf, ok := claim["payload"]
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "payload")
	}

	payload := et.Json{}
	switch v := payloadItf.(type) {
	case map[string]interface{}:
		payload = v
	case et.Json:
		payload = v
	}

	second, ok := claim["duration"].(float64)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "duration")
	}

	duration := time.Duration(second)
	result := &Claim{
		App:      app,
		Device:   device,
		Username: username,
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
