package utility

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

/**
* EncryptMD5 return a string with the value encrypted in md5
* @param value string
* @return string, error
**/
func EncryptMD5(value string) (string, error) {
	hash := md5.Sum([]byte(value))
	return hex.EncodeToString(hash[:]), nil
}

/**
* HashSHA1 return a string with the value encrypted in sha1
* @param value string
* @return string, error
**/
func HashSHA1(value string) (string, error) {
	hash := sha1.Sum([]byte(value))
	return hex.EncodeToString(hash[:]), nil
}

/**
* HashSHA256 return a string with the value encrypted in sha256
* @param value string
* @return string, error
**/
func HashSHA256(value string) (string, error) {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:]), nil
}

/**
* CryptoSHA512 return a string with the value encrypted in sha512
* @param value string
* @return string, error
**/
func HashSHA512(value string) (string, error) {
	hash := sha512.Sum512([]byte(value))
	return hex.EncodeToString(hash[:]), nil
}

/**
* EncryptAES return a string with the value encrypted in aes
* @param value string
* @return string, error
**/
func EncryptAES(value, secret string) (string, error) {
	data := []byte(value)
	key := []byte(secret)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	cipherText := make([]byte, aes.BlockSize+len(data))
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("could not encrypt: %v", err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], data)
	result := base64.StdEncoding.EncodeToString(cipherText)

	return result, nil
}

/**
* DecryptoAES return a string with the value decrypted in aes
* @param value string
* @return string, error
**/
func DecryptAES(value, secret string) (string, error) {
	key := []byte(secret)
	cipherText, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(cipherText) < aes.BlockSize {
		return "", err
	}

	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	return string(cipherText), nil

}
