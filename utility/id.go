package utility

import (
	cr "crypto/rand"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/google/uuid"
	"github.com/oklog/ulid"
)

/**
* GetOTP return a code verify
* @param length int
* @return string
**/
func GetOTP(length int) string {
	const charset = "0123456789"
	var seededRand *rand.Rand = rand.New(rand.NewSource(timezone.NowTime().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	return string(b)
}

/**
* UUID return a new UUID
* @return string
**/
func UUID() string {
	return uuid.NewString()
}

/**
* ULID return a new ULID
* @return string
**/
func ULID() string {
	t := time.Now().UTC()
	entropy := ulid.Monotonic(cr.Reader, 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)

	return id.String()
}

/**
* ShortUUID return a new ShortUUID
* @return string
**/
func Snowflake(nodeId int64, tag string) string {
	nano := strs.Format(`%d`, timezone.NowTime().UnixMilli())
	ns, err := mem.Get(nano, "-1")
	if err != nil {
		return RecordId(tag, "")
	}

	n, err := strconv.ParseInt(ns, 10, 64)
	if err != nil {
		return RecordId(tag, "")
	}

	n++
	mem.Set(nano, strconv.FormatInt(n, 10), 1)
	return strs.Format(`%s:%s%d%08v`, tag, nano, nodeId, n)
}

/**
* GenId return a new UUID
* @param id string
* @return string
**/
func GenId(id string) string {
	if map[string]bool{"": true, "*": true, "new": true}[id] {
		return UUID()
	}

	return id
}

/**
* GenKey return a new UUID
* @param id string
* @return string
**/
func GenKey(id string) string {
	if map[string]bool{"": true, "-1": true, "*": true, "new": true}[id] {
		return UUID()
	}

	return id
}

/**
* RecordId return a new UUID
* @param table string
* @param id string
* @return string
**/
func RecordId(tag, id string) string {
	if !map[string]bool{"": true, "*": true, "new": true}[id] {
		split := strings.Split(id, ":")
		if len(split) == 1 {
			return strs.Format(`%s:%s`, tag, id)
		} else if len(split) == 2 && split[0] != tag {
			return strs.Format(`%s:%s`, tag, split[1])
		}

		return id
	}

	id = UUID()
	return strs.Format(`%s:%s`, tag, id)
}

func init() {
	epoch := time.Date(2020, 1, 1, 0, 0, 0, 0, NowTime().Location())
	snowflake.Epoch = epoch.UnixNano() / 1e6
}
