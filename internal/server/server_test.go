package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sourcecd/gofermart/internal/models"
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

func TestUserParse(t *testing.T) {
	testCases := []struct {
		name   string
		req    string
		expAns *models.User
		expErr error
	}{
		{
			name: "OKReq",
			req:  `{"login": "test", "password": "testok"}`,
			expAns: &models.User{
				Login:    "test",
				Password: "testok",
			},
			expErr: nil,
		},
		{
			name:   "ErrReq1",
			req:    `{"login": "test", "password":`,
			expAns: &models.User{},
			expErr: prjerrors.ErrReqJSONParse,
		},
		{
			name:   "ErrReq2",
			req:    `{"login": "", "password":"qwe"}`,
			expAns: nil,
			expErr: prjerrors.ErrValidateLogPass,
		},
		{
			name:   "ErrReq3",
			req:    `{"login": "qwe", "password":""}`,
			expAns: nil,
			expErr: prjerrors.ErrValidateLogPass,
		},
	}

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			reader := strings.NewReader(v.req)
			testReq := httptest.NewRequest(http.MethodPost, "/", reader)

			user, err := userParse(testReq)

			require.ErrorIs(t, err, v.expErr)
			if err == nil {
				require.Equal(t, v.expAns.Login, user.Login)
				require.Equal(t, v.expAns.Password, user.Password)
			}
		})
	}
}

func TestSetCoockie(t *testing.T) {
	w := httptest.NewRecorder()

	SetTokenCookie(w, "test")

	resp := w.Result()
	cookie := resp.Cookies()
	require.Len(t, cookie, 1)
	require.Equal(t, cookie[0].Name, "Bearer")
	require.Equal(t, cookie[0].Value, "test")
}

func TestCheckContentType(t *testing.T) {
	testReq := httptest.NewRequest(http.MethodPost, "/", nil)
	testReq.Header.Set("Content-Type", "application/json")

	err := checkContentType(testReq, "application/json")
	require.NoError(t, err)
	err = checkContentType(testReq, "text/html")
	require.Error(t, err)
}
