package cryptandsign

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func GenRandKey() (*string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	encodedKey := hex.EncodeToString(key)
	return &encodedKey, nil
}

func GetPassHash(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	dst := h.Sum(nil)
	return hex.EncodeToString(dst)
}
