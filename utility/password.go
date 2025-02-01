package utility

import (
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

const (
	HASH_COST = 5
)

func PasswordHash(password string) (string, error) {
	var result string
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(password), HASH_COST)
	if err != nil {
		return result, err
	}

	result = string(hashPassword)

	return result, nil
}

func PasswordMatch(hashPassword string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(password))

	return err == nil
}

func PasswordSha256(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}
