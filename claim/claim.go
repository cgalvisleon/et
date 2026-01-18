package claim

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/golang-jwt/jwt/v4"
)

type ContextKey string

/**
* String
* @param ctx context.Context, def string
* @return string
**/
func (s ContextKey) String(ctx context.Context, def string) string {
	val := ctx.Value(s)
	result, ok := val.(string)
	if !ok {
		return def
	}

	return result
}

/**
* Duration
* @param ctx context.Context, def time.Duration
* @return time.Duration
**/
func (s ContextKey) Duration(ctx context.Context, def time.Duration) time.Duration {
	val := ctx.Value(s)
	result, ok := val.(time.Duration)
	if !ok {
		return def
	}

	return result
}

/**
* Json
* @param ctx context.Context, def et.Json
* @return et.Json
**/
func (s ContextKey) Json(ctx context.Context, def et.Json) et.Json {
	val := ctx.Value(s)
	result, ok := val.(et.Json)
	if !ok {
		return def
	}

	return result
}

const (
	DurationKey ContextKey = "duration"
	PayloadKey  ContextKey = "payload"
)

type Claim struct {
	jwt.StandardClaims
	Salt     string        `json:"salt"`
	Duration time.Duration `json:"duration"`
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
* GetTokenKey
* @param app, device, id string
* @return string
**/
func GetTokenKey(app, device, id string) string {
	return fmt.Sprintf("token:%s:%s:%s", app, device, id)
}

/**
* GenToken
* @param c Claim
* @return string, error
**/
func GenToken(c Claim, secret string) (string, error) {
	_jwt := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	result, err := _jwt.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return result, nil
}

/**
* newTokenKey
* @param c Claim
* @return string, error
**/
func newTokenKey(c Claim) (string, error) {
	secret := envar.GetStr("1977", "SECRET")
	result, err := GenToken(c, secret)
	if err != nil {
		return "", err
	}

	key := GetTokenKey(c.App, c.Device, c.ID)
	cache.SetDuration(key, result, c.Duration)

	return result, nil
}

/**
* NewToken
* @param id, app, name, username, device string, duration time.Duration
* @return string, string, error
**/
func NewToken(id, app, name, username, device string, duration time.Duration) (string, error) {
	c := NewClaim(id, app, name, username, device, "", "", "", duration)
	return newTokenKey(c)
}

/**
* NewAuthorization
* @param id, app, name, username, device string, tenantId, profileTp string, duration time.Duration
* @return string, error
**/
func NewAuthorization(id, app, name, username, device, tenantId, profileTp string, duration time.Duration) (string, error) {
	c := NewClaim(id, app, name, username, device, tenantId, profileTp, "", duration)
	return newTokenKey(c)
}

/**
* NewEphemeralToken
* @param id, app, name, username, device, tag string, duration time.Duration
* @return string, error
**/
func NewEphemeralToken(id, app, name, username, device, tag string, duration time.Duration) (string, error) {
	if tag == "" {
		return "", logs.Alertm("Tag is required")
	}

	if duration <= 0 {
		return "", logs.Alertm("Duration is required")
	}

	c := NewClaim(id, app, name, username, device, "", "", tag, duration)
	return newTokenKey(c)
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
* @param app, device, id string
* @return error
**/
func DeleteToken(app, device, id string) error {
	key := GetTokenKey(app, device, id)
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

	return DeleteToken(claim.App, claim.Device, claim.ID)
}

/**
* ParceToken
* @param token string
* @return *Claim, error
**/
func ParceToken(token string) (*Claim, error) {
	secret := envar.GetStr("1977", "SECRET")
	jToken, err := jwt.Parse(token, func(*jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, logs.Alert(err)
	}

	if !jToken.Valid {
		return nil, logs.Alertm(MSG_TOKEN_INVALID)
	}

	claim, ok := jToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, logs.Alertm(MSG_REQUIRED_INVALID)
	}

	app, ok := claim["app"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "app")
	}

	id, ok := claim["id"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "id")
	}

	name, ok := claim["name"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "name")
	}

	username, ok := claim["username"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "username")
	}

	device, ok := claim["device"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "device")
	}

	second, ok := claim["duration"].(float64)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "duration")
	}

	tenantId, ok := claim["tenantId"].(string)
	if !ok {
		tenantId = ""
	}

	profileTp, ok := claim["profileTp"].(string)
	if !ok {
		profileTp = ""
	}

	tag, ok := claim["tag"].(string)
	if !ok {
		tag = ""
	}

	duration := time.Duration(second)

	result := &Claim{
		ID:        id,
		App:       app,
		Name:      name,
		Username:  username,
		Device:    device,
		Duration:  duration,
		TenantId:  tenantId,
		ProfileTp: profileTp,
		Tag:       tag,
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

	key := GetTokenKey(result.App, result.Device, result.ID)
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
* @param app, device, id, token string, duration time.Duration
* @return error
**/
func SetToken(app, device, id, token string, duration time.Duration) error {
	key := GetTokenKey(app, device, id)
	if duration < 0 {
		cache.Delete(key)
		return fmt.Errorf(MSG_TOKEN_EXPIRED)
	}

	cache.Set(key, token, duration)

	return nil
}

/**
* ServiceId
* @param r *http.Request
* @return string
**/
func ServiceId(r *http.Request) string {
	ctx := r.Context()
	return ServiceIdKey.String(ctx, "-1")
}

/**
* ClientId
* @param r *http.Request
* @return string
**/
func ClientId(r *http.Request) string {
	ctx := r.Context()
	return ClientIdKey.String(ctx, "-1")
}

/**
* ClientName
* @param r *http.Request
* @return string
**/
func ClientName(r *http.Request) string {
	ctx := r.Context()
	return NameKey.String(ctx, "Anonimo")
}

/**
* Username
* @param r *http.Request
* @return string
**/
func Username(r *http.Request) string {
	ctx := r.Context()
	return UsernameKey.String(ctx, "Anonimo")
}

/**
* ProfileTp
* @param r *http.Request
* @return string
**/
func ProfileTp(r *http.Request) string {
	ctx := r.Context()
	return ProfileTpKey.String(ctx, "")
}

/**
* TenantId
* @param r *http.Request
* @return string
**/
func TenantId(r *http.Request) string {
	ctx := r.Context()
	return TenantIdKey.String(ctx, "")
}

/**
* Tag
* @param r *http.Request
* @return string
**/
func Tag(r *http.Request) string {
	ctx := r.Context()
	return TagKey.String(ctx, "")
}

/**
* Device
* @param r *http.Request
* @return string
**/
func Device(r *http.Request) string {
	ctx := r.Context()
	return DeviceKey.String(ctx, "")
}

/**
* GetClient
* @param r *http.Request
* @return et.Json
**/
func GetClient(r *http.Request) et.Json {
	now := timezone.NowTime()
	ctx := r.Context()
	clientId := ClientIdKey.String(ctx, "-1")
	serviceId := ServiceIdKey.String(ctx, "-1")
	app := AppKey.String(ctx, "")
	username := UsernameKey.String(ctx, "Anonimo")
	device := DeviceKey.String(ctx, "")
	fullName := NameKey.String(ctx, "Anonimo")
	profileTp := ProfileTpKey.String(ctx, "")
	tenantId := TenantIdKey.String(ctx, "")
	tag := TagKey.String(ctx, "")

	return et.Json{
		"date_at":    now,
		"client_id":  clientId,
		"service_id": serviceId,
		"app":        app,
		"username":   username,
		"device":     device,
		"full_name":  fullName,
		"profile_tp": profileTp,
		"tenant_id":  tenantId,
		"tag":        tag,
	}
}
