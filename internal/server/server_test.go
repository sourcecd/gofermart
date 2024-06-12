package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sourcecd/gofermart/internal/prjerrors"
	"github.com/stretchr/testify/require"
)

const expired = 30

func TestCheckRequestCreds(t *testing.T) {
	testCases := []struct {
		name   string
		cookie *http.Cookie
		expErr error
		expVal string
	}{
		{
			name: "okCoockie",
			cookie: &http.Cookie{
				Name:   "Bearer",
				Value:  "tokenOKqweewq",
				MaxAge: expired,
			},
			expErr: nil,
			expVal: "tokenOKqweewq",
		},
		{
			name:   "NoCoockie",
			cookie: &http.Cookie{},
			expErr: prjerrors.ErrAuthCredsNotFound,
			expVal: "",
		},
	}

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			testReq := httptest.NewRequest(http.MethodPost, "/", nil)
			testReq.AddCookie(v.cookie)

			res, err := checkRequestCreds(testReq)
			require.ErrorIs(t, err, v.expErr)
			require.Equal(t, v.expVal, res)
		})
	}
}

/*func checkRequestCreds(r *http.Request) (*string, error) {
	if ck, err := r.Cookie("Bearer"); err == nil {
		return &ck.Value, nil
	}
	if bearer := r.Header.Get("Authorization"); bearer != "" {
		headerSlice := strings.Split(bearer, " ")
		if len(headerSlice) == 2 && headerSlice[0] == "Bearer" {
			bearer = headerSlice[1]
			return &bearer, nil
		}
	}
	return nil, errors.New("auth creds not found")
}*/
