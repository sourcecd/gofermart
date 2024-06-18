package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func GenerateRandomKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	encodedKey := hex.EncodeToString(key)
	return encodedKey, nil
}

func GeneratePasswordHash(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	dst := h.Sum(nil)
	return hex.EncodeToString(dst)
}
