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
		testTime     = time.Now().Format(time.RFC3339)
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
				UploadedAt: testTime,
			})
			return nil
		})
	jsonExpRes := fmt.Sprintf(`
	[
		{
			"number": "12345678903",
			"status": "NEW",
			"uploaded_at": "%s"
		}
	]
	`, testTime)

	h := &handlers{
		ctx:    context.Background(),
		seckey: seckey,
		db:     db,
		rtr:    retr.NewRetr(),
	}

	//target check handler
	h.ordersList()(w, r)
	res := w.Result()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()
	require.JSONEq(t, jsonExpRes, string(b))
}

func TestGetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := mock.NewMockStore(ctrl)

	var balancePtr *models.Balance

	h := &handlers{
		ctx:    context.Background(),
		seckey: seckey,
		db:     db,
		rtr:    retr.NewRetr(),
	}

	db.EXPECT().GetBalance(gomock.Any(), userID, gomock.AssignableToTypeOf(balancePtr)).DoAndReturn(
		func(ctx context.Context, userid int64, balance *models.Balance) error {
			balance.Current = 10.5
			balance.Withdrawn = 42.5
			return nil
		})
	jsonExpRes := `
	{
		"current": 10.5,
		"withdrawn": 42.5
	}
	`

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{
		Name:  "Bearer",
		Value: tokenTest,
	})
	w := httptest.NewRecorder()

	//target check handler
	h.getBalance()(w, r)

	res := w.Result()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()

	require.JSONEq(t, jsonExpRes, string(b))
}

func TestWithdraw(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := mock.NewMockStore(ctrl)

	jsonReq := `{"order":"12345678903", "sum":10.5}`

	h := &handlers{
		ctx:    context.Background(),
		seckey: seckey,
		db:     db,
		rtr:    retr.NewRetr(),
	}

	db.EXPECT().Withdraw(gomock.Any(), userID, &models.Withdraw{Order: "12345678903", Sum: 10.5}).Return(nil)

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonReq))
	r.Header.Add("Content-Type", "application/json")
	r.AddCookie(&http.Cookie{
		Name:  "Bearer",
		Value: tokenTest,
	})
	w := httptest.NewRecorder()

	//target check header
	h.withdraw()(w, r)

	res := w.Result()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "\n", string(b))
}

func TestWithdrawals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := mock.NewMockStore(ctrl)

	var withdrawalsPtr *[]models.Withdrawals
	procTime := time.Now().Format(time.RFC3339)

	h := &handlers{
		ctx:    context.Background(),
		seckey: seckey,
		db:     db,
		rtr:    retr.NewRetr(),
	}

	db.EXPECT().Withdrawals(gomock.Any(), userID, gomock.AssignableToTypeOf(withdrawalsPtr)).DoAndReturn(
		func(ctx context.Context, userid int64, withdrawals *[]models.Withdrawals) error {
			*withdrawals = append(*withdrawals, models.Withdrawals{
				Order:       "12345678903",
				Sum:         15.5,
				ProcessedAt: procTime,
			})
			return nil
		})
	jsonAndRes := fmt.Sprintf(`[{"order": "12345678903", "sum": 15.5, "processed_at": "%s"}]`, procTime)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{
		Name:  "Bearer",
		Value: tokenTest,
	})
	w := httptest.NewRecorder()

	//target check header
	h.withdrawals()(w, r)

	res := w.Result()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()
	require.JSONEq(t, jsonAndRes, string(b))
}
