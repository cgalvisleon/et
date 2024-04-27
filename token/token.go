package token

import (
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
	"github.com/golang-jwt/jwt/v4"
)

type Token struct {
	secret string
	cache  cache.Cache
}

var conn *Token

func Load(cache cache.Cache) (*Token, error) {
	if conn != nil {
		return conn, nil
	}

	conn = &Token{
		secret: envar.EnvarStr("", "SECRET"),
		cache:  cache,
	}

	return conn, nil
}

func (t *Token) genToken(sub, name, app, kind, device string, expired time.Duration) (token, key string, err error) {
	id := utility.UUID()
	c := Claim{
		Id:     id,
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
	key = TokenKey(app, device, id)

	return
}
