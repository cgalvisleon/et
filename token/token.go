package token

import (
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/pubsub"
	"github.com/golang-jwt/jwt/v4"
)

type Token struct {
	secret string
	cache  cache.Cache
	pubsub pubsub.PubSub
}

var conn *Token

func Load(cache cache.Cache, pubsub pubsub.PubSub) (*Token, error) {
	if conn != nil {
		return conn, nil
	}

	conn = &Token{
		secret: envar.EnvarStr("", "SECRET"),
		cache:  cache,
		pubsub: pubsub,
	}

	return conn, nil
}

func (t *Token) generate(sub, name, app, kind, device string, expired time.Duration) (token, key string, err error) {
	c := Claim{
		Sub:    sub,
		Name:   name,
		Iat:    time.Now(),
		Exp:    expired,
		App:    app,
		Kind:   kind,
		Device: device,
	}
	_jwt := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	token, err = _jwt.SignedString([]byte(t.secret))
	if err != nil {
		return
	}
	key = TokenKey(app, device, sub)

	return
}

func (t *Token) parce(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(*jwt.Token) (interface{}, error) {
		return []byte(t.secret), nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}
