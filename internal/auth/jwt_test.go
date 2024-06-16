package auth

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	secKey       = "zoku4eitieC6meingu4xoh3tiePh4sei"
	lenToken     = 121
	userID       = int64(10)
	expiredToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTgzMDk4NDcsIlVzZXJJRCI6MTB9.xtyunCHNAG5iun28LmDGJUD6zQSsMsd3Y13tQe66QQ4"
)

var testToken string

func TestGenerateJWT(t *testing.T) {
	token, err := GenerateJWT(userID, secKey)
	require.NoError(t, err)
	assert.Len(t, token, lenToken)
	testToken = token
}

func TestExtractJWT(t *testing.T) {
	testCases := []struct {
		name      string
		token     string
		expUserID int64
		expErr    error
	}{
		{
			name:      "tokenOK",
			token:     testToken,
			expUserID: userID,
			expErr:    nil,
		},
		{
			name:      "tokenExp",
			token:     expiredToken,
			expUserID: -1,
			expErr:    errors.New("expectedError"),
		},
		{
			name:      "wrongToken",
			token:     "fakejwt",
			expUserID: -1,
			expErr:    errors.New("expectedError"),
		},
	}

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			userid, err := ParseJWT(v.token, secKey)
			if v.expErr != nil {
				require.Error(t, err)
			}
			if v.expErr == nil {
				require.NoError(t, err)
			}
			assert.Equal(t, v.expUserID, userid)
		})
	}
}
