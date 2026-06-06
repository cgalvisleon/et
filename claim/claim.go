package claim

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/golang-jwt/jwt/v4"
)

var (
	jwtSecret     string
	jwtSecretOnce sync.Once
)

/**
* getSecret returns the JWT signing secret, reading the SECRET env var only once.
* @return string
**/
func getSecret() string {
	secret := envar.GetStr("SECRET", "1977")
	if config.IsLoad() {
		secret = config.GetStr("SECRET", secret)
	}

	jwtSecretOnce.Do(func() {
		jwtSecret = secret
	})
	return jwtSecret
}

/**
* Claim JWT payload with standard claims and application-specific fields.
* tenantId is stored inside Payload; use claim.TenantId(r) to extract it.
**/
type Claim struct {
	jwt.StandardClaims
	ID        string        `json:"id"`
	Salt      string        `json:"salt"`
	Duration  time.Duration `json:"duration"`
	App       string        `json:"app"`
	Device    string        `json:"device"`
	UserId    string        `json:"userId"`
	Username  string        `json:"username"`
	TenantId  string        `json:"tenantId"`
	ProfileId string        `json:"profileId"`
	Payload   et.Json       `json:"payload"`
}

/**
* ToJson
* @return et.Json
**/
func (s *Claim) ToJson() (et.Json, error) {
	result := et.Json{
		"id":        s.ID,
		"salt":      s.Salt,
		"duration":  s.Duration,
		"app":       s.App,
		"device":    s.Device,
		"userId":    s.UserId,
		"username":  s.Username,
		"tenantId":  s.TenantId,
		"profileId": s.ProfileId,
		"payload":   s.Payload,
		"expiresAt": time.Unix(s.ExpiresAt, 0).Format("2006-01-02 03:04:05 PM"),
	}
	if s.ExpiresAt != 0 {
		result["exp"] = s.ExpiresAt
	}
	if s.Issuer != "" {
		result["iss"] = s.Issuer
	}
	if s.Subject != "" {
		result["sub"] = s.Subject
	}
	if s.Audience != "" {
		result["aud"] = s.Audience
	}
	if s.IssuedAt != 0 {
		result["iat"] = s.IssuedAt
	}
	if s.NotBefore != 0 {
		result["nbf"] = s.NotBefore
	}
	if s.Id != "" {
		result["jti"] = s.Id
	}
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
	result.ID = reg.ULID()
	result.Salt = utility.GetOTP(6)
	result.Duration = duration
	if result.Duration != 0 {
		result.ExpiresAt = timezone.Add(result.Duration).Unix()
	}

	return result
}

/**
* genToken
* @param c *Claim, secret string
* @return string, error
**/
func genToken(c *Claim, secret string) (string, error) {
	_jwt := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	result, err := _jwt.SignedString([]byte(secret))
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
	secret := getSecret()
	jToken, err := jwt.Parse(token, func(*jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if !jToken.Valid {
		return nil, errors.New(msg.MSG_TOKEN_INVALID)
	}

	claim, ok := jToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New(msg.MSG_REQUIRED_INVALID)
	}

	id, ok := claim["id"].(string)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "id")
	}

	app, ok := claim["app"].(string)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "app")
	}

	device, ok := claim["device"].(string)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "device")
	}

	userId, ok := claim["userId"].(string)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "userId")
	}

	username, ok := claim["username"].(string)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "username")
	}

	second, ok := claim["duration"].(float64)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "duration")
	}

	payloadItf, ok := claim["payload"]
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "payload")
	}

	tenantId, ok := claim["tenantId"].(string)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "tenantId")
	}

	profileId, ok := claim["profileId"].(string)
	if !ok {
		return nil, fmt.Errorf(msg.MSG_TOKEN_INVALID_ATRIB, "profileId")
	}

	payload := et.Json{}
	switch v := payloadItf.(type) {
	case map[string]interface{}:
		payload = v
	case et.Json:
		payload = v
	}

	duration := time.Duration(second)
	result := &Claim{
		ID:        id,
		App:       app,
		Device:    device,
		UserId:    userId,
		Username:  username,
		Duration:  duration,
		TenantId:  tenantId,
		ProfileId: profileId,
		Payload:   payload,
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
* NewToken
* @param app, device, userId, username, tenantId, profileId string, payload et.Json, duration time.Duration
* @return string, error
**/
func NewToken(app, device, userId, username, tenantId, profileId string, payload et.Json, duration time.Duration) (string, error) {
	if app == "" {
		return "", fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "app")
	}
	c := NewClaim(duration)
	c.App = app
	c.Device = device
	c.UserId = userId
	c.Username = username
	c.TenantId = tenantId
	c.ProfileId = profileId
	c.Payload = payload
	result, err := genToken(c, getSecret())
	if err != nil {
		return "", err
	}

	return result, nil
}
