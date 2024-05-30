package cryptandsign

import (
	"crypto/rand"
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
