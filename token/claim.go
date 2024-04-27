package token

import (
	"context"
	"net/http"
	"time"

	"github.com/cgalvisleon/elvis/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/generic"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/golang-jwt/jwt/v4"
)

type Claim struct {
	Id     string        `json:"id"`
	Sub    string        `json:"sub"`
	Name   string        `json:"name"`
	Iat    time.Time     `json:"iat"`
	Exp    time.Duration `json:"exp"`
	App    string        `json:"app"`
	Kind   string        `json:"kind"`
	Device string        `json:"device"`
	jwt.StandardClaims
}

func DelToken(app, device, id string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_TOKEN_SERVICE)
	}

	key := TokenKey(app, device, id)
	ok := conn.cache.Del(key)

	event.Publish(key, "token/delete", et.Json{
		"key": key,
	})

	return nil
}

func DelTokeStrng(tokenString string) error {
	secret := envar.EnvarStr("", "SECRET")
	token, err := jwt.Parse(tokenString, func(*jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return err
	}

	claim, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil
	}

	app, ok := claim["app"].(string)
	if !ok {
		return nil
	}

	id, ok := claim["id"].(string)
	if !ok {
		return nil
	}

	device, ok := claim["device"].(string)
	if !ok {
		return nil
	}

	ctx := context.Background()
	return DelTokenCtx(ctx, app, device, id)
}

func TokenKey(app, device, id string) string {
	str := strs.Append(app, device, "-")
	str = strs.Append(str, id, "-")
	return strs.Format(`token:%s`, str)
}

func ParceToken(tokenString string) (*Claim, error) {
	secret := envar.EnvarStr("", "SECRET")
	token, err := jwt.Parse(tokenString, func(*jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, logs.Alertm(MSG_TOKEN_INVALID)
	}

	claim, ok := token.Claims.(jwt.MapClaims)
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

	kind, ok := claim["kind"].(string)
	if !ok {
		return nil, logs.Alertf(MSG_TOKEN_INVALID_ATRIB, "kind")
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

	duration := time.Duration(second)

	return &Claim{
		ID:       id,
		App:      app,
		Name:     name,
		Kind:     kind,
		Username: username,
		Device:   device,
		Duration: duration,
	}, nil
}

func GetFromToken(ctx context.Context, tokenString string) (*Claim, error) {
	result, err := ParceToken(tokenString)
	if err != nil {
		return nil, err
	}

	key := TokenKey(result.App, result.Device, result.ID)
	c, err := cache.GetCtx(ctx, key, "")
	if err != nil {
		return nil, logs.Alertm(MSG_TOKEN_INVALID)
	}

	if c != tokenString {
		return nil, logs.Alertm(MSG_TOKEN_INVALID)
	}

	err = cache.SetCtx(ctx, key, c, result.Duration)
	if err != nil {
		return nil, logs.Alertm(MSG_TOKEN_INVALID)
	}

	return result, nil
}

func GenTokenCtx(ctx context.Context, id, app, name, kind, username, device string, duration time.Duration) (string, error) {
	token, key, err := genToken(id, app, name, kind, username, device, duration)
	if err != nil {
		return "", err
	}

	err = cache.SetCtx(ctx, key, token, duration)
	if err != nil {
		return "", err
	}

	event.Publish(key, "token/create", et.Json{
		"key":  key,
		"toke": token,
	})

	return token, nil
}

func GenToken(id, app, name, kind, username, device string, duration time.Duration) (string, error) {
	ctx := context.Background()
	return GenTokenCtx(ctx, id, app, name, kind, username, device, duration)
}

func GetClient(r *http.Request) et.Json {
	now := utility.Now()
	ctx := r.Context()

	return et.Json{
		"date_of":   now,
		"client_id": generic.New(ctx.Value("clientId")).Str(),
		"username":  generic.New(ctx.Value("username")).Str(),
		"name":      generic.New(ctx.Value("name")).Str(),
	}
}
