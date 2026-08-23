package utils

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"

	"github.com/mervick/aes-everywhere/go/aes256"
)

func TestDecodeDecryptSurvivesAlreadyUnescapedCiphertext(t *testing.T) {
	// A form-encoded or query-string request reaches the handler already
	// unescaped, so the same ciphertext has to decrypt in both shapes.
	for attempt := 0; attempt < 200; attempt++ {
		raw := aes256.Encrypt("device-password", SecretKey)
		escaped := url.QueryEscape(raw)
		for _, wire := range []string{raw, escaped} {
			password, err := DecodeDecrypt(wire)
			if err != nil || password != "device-password" {
				t.Fatalf("DecodeDecrypt(%q) = %q, %v", wire, password, err)
			}
		}
		if strings.ContainsAny(raw, "+/=") && attempt > 0 {
			return
		}
	}
}

func TestDecodeDecryptRejectsMalformedCiphertext(t *testing.T) {
	malformed := []string{"", "not base64 %%%", "%zz"}
	for _, length := range []int{8, 16, 20, 32, 48} {
		envelope := make([]byte, length)
		copy(envelope, saltedPrefix)
		for index := 8; index < length; index++ {
			envelope[index] = byte(index)
		}
		malformed = append(malformed, base64.StdEncoding.EncodeToString(envelope))
	}
	for _, wire := range malformed {
		password, err := DecodeDecrypt(wire)
		if err == nil || password != "" {
			t.Fatalf("DecodeDecrypt(%q) = %q, %v; want an error", wire, password, err)
		}
	}
}
