package utility

import (
	cr "crypto/rand"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/google/uuid"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/oklog/ulid"
)

var nodId int64 = 1

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
* NanoId return a new NanoID
* @return string
**/
func NanoId(n int) string {
	id, err := gonanoid.New(n)
	if err != nil {
		panic(err)
	}

	return id
}

/**
* ShortUUID return a new ShortUUID
* @return string
**/
func Snowflake() string {
	node, err := snowflake.NewNode(nodId)
	if err != nil {
		panic(err)
	}
	id := node.Generate()

	return id.String()
}

/**
* SetSnowflakeNode
* @param n int64
**/
func SetSnowflakeNode(n int64) {
	nodId = n
}

/**
* PrefixId return a new UUID with prefix
* @param prefix string
* @return string
**/
func PrefixId(prefix string) string {
	id := Snowflake()
	return prefix + "-" + id
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
func RecordId(table, id string) string {
	if !map[string]bool{"": true, "*": true, "new": true}[id] {
		split := strings.Split(id, ":")
		if len(split) == 1 {
			return strs.Format(`%s:%s`, table, id)
		} else if len(split) == 2 && split[0] != table {
			return strs.Format(`%s:%s`, table, split[1])
		}

		return id
	}

	id = Snowflake()
	return strs.Format(`%s:%s`, table, id)
}

func init() {
	epoch := time.Date(2020, 1, 1, 0, 0, 0, 0, NowTime().Location())
	snowflake.Epoch = epoch.UnixNano() / 1e6
}
