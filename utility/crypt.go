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

	"github.com/cgalvisleon/et/envar"
)

type CryptoType int

const (
	MD5 CryptoType = iota
	SHA1
	SHA256
	SHA512
	AES
)

// String return string of crypto type
func (c CryptoType) String() string {
	switch c {
	case MD5:
		return "MD5"
	case SHA1:
		return "SHA1"
	case SHA256:
		return "SHA256"
	case SHA512:
		return "SHA512"
	case AES:
		return "AES"
	}
	return ""
}

// Type return a crypto type from a string
func GetCryptoType(value string) CryptoType {
	switch value {
	case "MD5":
		return MD5
	case "SHA1":
		return SHA1
	case "SHA256":
		return SHA256
	case "SHA512":
		return SHA512
	case "AES":
		return AES
	}
	return MD5
}

// CryptoMD5 return a string with the value encrypted in md5
func cryptoMD5(value string) (string, error) {
	hash := md5.Sum([]byte(value))
	return hex.EncodeToString(hash[:]), nil
}

// CryptoSHA1 return a string with the value encrypted in sha1
func cryptoSHA1(value string) (string, error) {
	hash := sha1.Sum([]byte(value))
	return hex.EncodeToString(hash[:]), nil
}

// CryptoSHA256 return a string with the value encrypted in sha256
func cryptoSHA256(value string) (string, error) {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:]), nil
}

// CryptoSHA512 return a string with the value encrypted in sha512
func cryptoSHA512(value string) (string, error) {
	hash := sha512.Sum512([]byte(value))
	return hex.EncodeToString(hash[:]), nil
}

// CryptoAES return a string with the value encrypted in aes
func cryptoAES(value string) (string, error) {
	secret := envar.GetStr("", "SECRET")
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

func Encrypt(value string, cryptoType CryptoType) (string, error) {
	switch cryptoType {
	case MD5:
		return cryptoMD5(value)
	case SHA1:
		return cryptoSHA1(value)
	case SHA256:
		return cryptoSHA256(value)
	case SHA512:
		return cryptoSHA512(value)
	case AES:
		return cryptoAES(value)
	}
	return "", fmt.Errorf("crypto type not found")

}

func DecryptoAES(value string) (string, error) {
	secret := envar.GetStr("", "SECRET")
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
