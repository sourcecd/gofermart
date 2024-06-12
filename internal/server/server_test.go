package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sourcecd/gofermart/internal/prjerrors"
	"github.com/stretchr/testify/require"
)

const expired = 30

type authHeader struct {
	key,
	value string
}

func TestCheckRequestCreds(t *testing.T) {
	testCases := []struct {
		name       string
		cookie     *http.Cookie
		authHeader authHeader
		expErr     error
		expVal     string
	}{
		{
			name: "okCoockie",
			cookie: &http.Cookie{
				Name:   "Bearer",
				Value:  "tokenOKqweewq",
				MaxAge: expired,
			},
			authHeader: authHeader{},
			expErr:     nil,
			expVal:     "tokenOKqweewq",
		},
		{
			name:       "NoCoockie",
			cookie:     &http.Cookie{},
			authHeader: authHeader{},
			expErr:     prjerrors.ErrAuthCredsNotFound,
			expVal:     "",
		},
		{
			name:   "NoCoockieWithAuthHeader",
			cookie: &http.Cookie{},
			authHeader: authHeader{
				key:   "Authorization",
				value: "Bearer tokenOKheader",
			},
			expErr: nil,
			expVal: "tokenOKheader",
		},
	}

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			testReq := httptest.NewRequest(http.MethodPost, "/", nil)
			testReq.AddCookie(v.cookie)
			testReq.Header.Set(v.authHeader.key, v.authHeader.value)

			res, err := checkRequestCreds(testReq)
			require.ErrorIs(t, err, v.expErr)
			require.Equal(t, v.expVal, res)
		})
	}
}
