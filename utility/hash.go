package utility

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

const (
	HASH_COST = 5
)

/**
* Hash using bcrypt
* @param password string
* @return string, error
**/
func Hash(password string) (string, error) {
	var result string
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(password), HASH_COST)
	if err != nil {
		return result, err
	}

	result = string(hashPassword)

	return result, nil
}

/**
* Match using bcrypt
* @param hashPassword, password string
* @return bool
**/
func Match(hashPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(password))

	return err == nil
}

/**
* Hash using sha256
* @param password string
* @return string
**/
func Sha256(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

/**
* Hash using md5
* @param password string
* @return string
**/
func Md5(password string) string {
	hash := md5.Sum([]byte(password))
	return hex.EncodeToString(hash[:])
}
