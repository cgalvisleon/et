package reg

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/google/uuid"
	"github.com/oklog/ulid"
	"github.com/rs/xid"
)

/**
* UUID
* @return string
**/
func UUID() string {
	return uuid.NewString()
}

/**
* GenKey
* @params args ...interface{}
* @return string
**/
func GenKey(args ...interface{}) string {
	var keys []string
	for _, arg := range args {
		keys = append(keys, strs.Format(`%v`, arg))
	}

	return strings.Join(keys, ":")
}

/**
* Id
* @params tag string
* @return string
**/
func GenId(tag string) string {
	return strs.Format(`%s:%s`, tag, UUID())
}

/**
* GetId
* @params tag, id string
* @return string
**/
func GetId(tag, id string) string {
	if !map[string]bool{"": true, "*": true, "new": true}[id] {
		return id
	}

	return GenId(tag)
}

/**
* GenIndex
* @return int64
**/
func GenIndex() int64 {
	return timezone.NowTime().UnixNano()
}

/**
* GenSnowflake
* @return string
**/
func GenSnowflake() string {
	ms := timezone.NowTime().UnixMilli()
	return fmt.Sprintf("%d%03d", ms, rand.Intn(1000))
}

/**
* ULID
* @return string
**/
func ULID() string {
	t := timezone.NowTime()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

/**
* XID
* @return string
**/
func XID() string {
	id := xid.New()
	return id.String()
}

func init() {
	epoch := time.Date(2020, 1, 1, 0, 0, 0, 0, timezone.NowTime().Location())
	snowflake.Epoch = epoch.UnixNano() / 1e6
}
