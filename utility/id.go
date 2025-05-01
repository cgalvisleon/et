package utility

import (
	"math/rand"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/cgalvisleon/et/timezone"
	"github.com/google/uuid"
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
* UUID
* @return string
**/
func UUID() string {
	return uuid.NewString()
}

/**
* GenId
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
* GenKey
* @param id string
* @return string
**/
func GenKey(id string) string {
	if map[string]bool{"": true, "-1": true, "*": true, "new": true}[id] {
		return UUID()
	}

	return id
}

func init() {
	epoch := time.Date(2020, 1, 1, 0, 0, 0, 0, NowTime().Location())
	snowflake.Epoch = epoch.UnixNano() / 1e6
}
