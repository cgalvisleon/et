package utility

import (
	"math/rand"

	"github.com/cgalvisleon/et/timezone"
	"github.com/google/uuid"
)

/**
* GetRandom return a random string
* @param charset string, length int
* @return string
**/
func GetRandom(charset string, length int) string {
	var seededRand *rand.Rand = rand.New(rand.NewSource(timezone.NowTime().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	return string(b)
}

/**
* GetOTP return a code verify
* @param length int
* @return string
**/
func GetOTP(length int) string {
	const charset = "0123456789"
	return GetRandom(charset, length)
}

/**
* GetRandomString return a random string
* @param length int
* @return string
**/
func GetRandomString(length int) string {
	return GetRandom("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", length)
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
