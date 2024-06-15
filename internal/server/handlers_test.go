package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sourcecd/gofermart/internal/auth"
	"github.com/sourcecd/gofermart/internal/models"
	"github.com/sourcecd/gofermart/internal/retr"
	"github.com/sourcecd/gofermart/internal/storage/mock"
	"github.com/stretchr/testify/require"
)

const (
	seckey   = "oivohfo8Saelahv2vei8ee8Ighae3ei0"
	userID   = int64(100)
	tokenLen = 123
	login    = "test"
	password = "testpass"
)

var tokenTest string

func TestRegisterUser(t *testing.T) {
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

	//target test handler
	h.registerUser()(w, r)
	res := w.Result()

	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Len(t, b, tokenLen)

	//check
	userid, err := auth.ExtractJWT(string(b), h.seckey)
	require.NoError(t, err)
	require.Equal(t, userID, userid)
}

func TestAuthUser(t *testing.T) {
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

	db.EXPECT().AuthUser(gomock.Any(), &models.User{Login: login, Password: password}).Return(userID, nil)

	//target test handler
	h.authUser()(w, r)
	res := w.Result()

	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()
	tokenTest = string(b)
	require.Len(t, b, tokenLen)

	//check
	userid, err := auth.ExtractJWT(string(b), h.seckey)
	require.NoError(t, err)
	require.Equal(t, userID, userid)
}

func TestOrderRegister(t *testing.T) {
	orderNum := "12345678903"
	orderNumMock := int64(12345678903)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := mock.NewMockStore(ctrl)

	reader := strings.NewReader(orderNum)
	r := httptest.NewRequest(http.MethodPost, "/", reader)
	r.AddCookie(&http.Cookie{
		Name:  "Bearer",
		Value: tokenTest,
	})
	r.Header.Add("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	db.EXPECT().CreateOrder(gomock.Any(), userID, orderNumMock).Return(nil)

	h := &handlers{
		ctx:    context.Background(),
		seckey: seckey,
		db:     db,
		rtr:    retr.NewRetr(),
	}

	//target test handler
	h.orderRegister()(w, r)

	res := w.Result()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, orderNum, string(b))
}

func TestOrdersList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := mock.NewMockStore(ctrl)

	var (
		orderListPtr *[]models.Order
		testTime     = time.Now()
	)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{
		Name:  "Bearer",
		Value: tokenTest,
	})
	w := httptest.NewRecorder()

	db.EXPECT().ListOrders(gomock.Any(), userID, gomock.AssignableToTypeOf(orderListPtr)).DoAndReturn(
		func(ctx context.Context, userid int64, ord *[]models.Order) error {
			*ord = append(*ord, models.Order{
				Number:     "12345678903",
				Status:     "NEW",
				UploadedAt: testTime.Format(time.RFC3339),
			})
			return nil
		})
	jsonRes := fmt.Sprintf(`
	[
		{
			"number": "12345678903",
			"status": "NEW",
			"uploaded_at": "%s"
		}
	]
	`, testTime.Format(time.RFC3339))

	h := &handlers{
		ctx:    context.Background(),
		seckey: seckey,
		db:     db,
		rtr:    retr.NewRetr(),
	}

	h.ordersList()(w, r)
	res := w.Result()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()
	require.JSONEq(t, jsonRes, string(b))
}
