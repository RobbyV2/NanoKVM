package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"net/url"

	log "github.com/sirupsen/logrus"
)

// SecretKey is only used to prevent the data from being transmitted in plaintext.
const SecretKey = "nanokvm-sipeed-2024"

const (
	saltedPrefix = "Salted__"
	headerLength = 16
)

var ErrInvalidCiphertext = errors.New("invalid ciphertext")

// Decrypt reads the OpenSSL "Salted__" envelope that CryptoJS produces. Every
// length and padding byte is checked here because the ciphertext is attacker
// controlled and the upstream aes256 package slices it unchecked.
func Decrypt(ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	if len(raw) < headerLength+aes.BlockSize || !bytes.HasPrefix(raw, []byte(saltedPrefix)) || len(raw)%aes.BlockSize != 0 {
		return "", ErrInvalidCiphertext
	}

	key, iv := deriveKeyAndIV(SecretKey, raw[len(saltedPrefix):headerLength])
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plaintext := make([]byte, len(raw)-headerLength)
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, raw[headerLength:])

	padding := int(plaintext[len(plaintext)-1])
	if padding < 1 || padding > aes.BlockSize || padding > len(plaintext) {
		return "", ErrInvalidCiphertext
	}
	for _, b := range plaintext[len(plaintext)-padding:] {
		if int(b) != padding {
			return "", ErrInvalidCiphertext
		}
	}
	return string(plaintext[:len(plaintext)-padding]), nil
}

// PathUnescape, not QueryUnescape: a form-encoded or query-string request has
// already been unescaped by the HTTP layer, and turning the '+' of the base64
// alphabet into a space there corrupted the ciphertext.
func DecodeDecrypt(data string) (string, error) {
	ciphertext, err := url.PathUnescape(data)
	if err != nil {
		log.Errorf("decode ciphertext failed: %s", err)
		return "", err
	}

	return Decrypt(ciphertext)
}

func deriveKeyAndIV(passphrase string, salt []byte) ([]byte, []byte) {
	var material, digest []byte
	for len(material) < 48 {
		sum := md5.Sum(append(append(append([]byte(nil), digest...), passphrase...), salt...))
		digest = sum[:]
		material = append(material, digest...)
	}
	return material[:32], material[32:48]
}
