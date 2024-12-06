package utility

import (
	cr "crypto/rand"
	"math/rand"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/cgalvisleon/et/timezone"
	"github.com/google/uuid"
	gonanoid "github.com/matoous/go-nanoid/v2"
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
func ShortUUID() string {
	node, err := snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}
	id := node.Generate()

	return id.String()
}

/**
* SnowflakeID return a new SnowflakeID
* @return string
**/
func SnowflakeID() string {
	node, err := snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}
	id := node.Generate()

	return id.String()
}

/**
* PrefixId return a new UUID with prefix
* @param prefix string
* @return string
**/
func PrefixId(prefix string) string {
	id := ShortUUID()
	return prefix + "-" + id
}

/**
* NewId return a new UUID
* @return string
**/
func NewId() string {
	return UUID()
}

/**
* GenId return a new UUID
* @param id string
* @return string
**/
func GenId(id string) string {
	if map[string]bool{"": true, "*": true, "new": true}[id] {
		return NewId()
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
