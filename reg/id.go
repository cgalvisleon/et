package reg

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/rs/xid"
)

/**
* UUID
* @return string
**/
func UUID() string {
	result, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return result.String()
}

/**
* ULID: Un ULID ordenable por tiempo que no se repite en el proceso: ulid.Make usa una
* entropía monotónica compartida y segura entre goroutines (antes se sembraba con el
* reloj en cada llamada y dos llamadas en el mismo instante daban el mismo valor)
* @return string
**/
func ULID() string {
	return ulid.Make().String()
}

/**
* XID
* @return string
**/
func XID() string {
	id := xid.New()
	return id.String()
}

/**
* GenKey
* @params args ...interface{}
* @return string
**/
func GenKey(args ...interface{}) string {
	var keys []string
	for _, arg := range args {
		keys = append(keys, fmt.Sprintf(`%v`, arg))
	}

	return strings.Join(keys, ":")
}

// GenUUId: Deprecated: use GenUUID.
func GenUUId(tag string) string {
	return GenUUID(tag)
}

/**
* GenUUID
* @params tag string
* @return string
**/
func GenUUID(tag string) string {
	return fmt.Sprintf(`%s:%s`, tag, UUID())
}

/**
* GenULID: tag:ULID (Universally Unique Lexicographically Sortable Identifier)
* @params tag string
* @return string
**/
func GenULID(tag string) string {
	return fmt.Sprintf(`%s:%s`, tag, ULID())
}

/**
* GenXID
* @params tag string
* @return string
**/
func GenXID(tag string) string {
	return fmt.Sprintf(`%s:%s`, tag, XID())
}

/**
* GetUUID
* @params id string
* @return string
**/
func GetUUID(id string) string {
	if !map[string]bool{"": true, "*": true, "new": true}[id] {
		return id
	}

	return UUID()
}

/**
* GetULID
* @params id string
* @return string
**/
func GetULID(id string) string {
	if !map[string]bool{"": true, "*": true, "new": true}[id] {
		return id
	}

	return ULID()
}

/**
* GetXID
* @params id string
* @return string
**/
func GetXID(id string) string {
	if !map[string]bool{"": true, "*": true, "new": true}[id] {
		return id
	}

	return XID()
}

/**
* GenIndex
* @return int64
**/
func GenIndex() int64 {
	return timezone.Now().UnixNano()
}

/**
* GenSnowflake
* @return string
**/
func GenSnowflake() string {
	ms := timezone.Now().UnixMilli()
	return fmt.Sprintf("%d%03d", ms, rand.Intn(1000))
}

/**
* GenHashKey
* @params args ...interface{}
* @return string
**/
func GenHashKey(args ...interface{}) string {
	key := GenKey(args...)
	return utility.ToBase64(key)
}

/**
* TagUUID
* @params tag, id string
* @return string
**/
func TagUUID(tag, id string) string {
	if map[string]bool{"": true, "*": true, "new": true}[id] {
		return fmt.Sprintf(`%s:%s`, tag, UUID())
	}

	return id
}

/**
* TagULID
* @params tag, id string
* @return string
**/
func TagULID(tag, id string) string {
	if map[string]bool{"": true, "*": true, "new": true}[id] {
		return fmt.Sprintf(`%s:%s`, tag, ULID())
	}

	return id
}

/**
* TagXID
* @params tag, id string
* @return string
**/
func TagXID(tag, id string) string {
	if map[string]bool{"": true, "*": true, "new": true}[id] {
		return fmt.Sprintf(`%s:%s`, tag, XID())
	}

	return id
}

func init() {
	epoch := time.Date(2020, 1, 1, 0, 0, 0, 0, timezone.Now().Location())
	snowflake.Epoch = epoch.UnixNano() / 1e6
}
