package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sourcecd/gofermart/internal/auth"
	"github.com/sourcecd/gofermart/internal/models"
	"github.com/sourcecd/gofermart/internal/retr"
	"github.com/sourcecd/gofermart/internal/storage/mock"
	"github.com/stretchr/testify/require"
)

const seckey = "oivohfo8Saelahv2vei8ee8Ighae3ei0"

func TestRegisterUser(t *testing.T) {
	userID := int64(10)
	login := "test"
	password := "testpass"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := mock.NewMockStore(ctrl)

	reader := strings.NewReader(fmt.Sprintf(`{"login": "%s", "password": "%s"}`, login, password))
	r := httptest.NewRequest(http.MethodPost, "/", reader)
	r.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h := &handlers{
		ctx:    context.Background(),
		seckey: seckey,
		db:     db,
		rtr:    retr.NewRetr(),
	}

	db.EXPECT().RegisterUser(gomock.Any(), &models.User{Login: login, Password: password}).Return(userID, nil)

	h.registerUser()(w, r)

	b, _ := io.ReadAll(w.Result().Body)
	defer w.Result().Body.Close()
	require.Len(t, b, 121)

	userid, _ := auth.ExtractJWT(string(b), h.seckey)
	require.Equal(t, userID, userid)
}
