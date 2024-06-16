package storage

import (
	"context"

	"github.com/sourcecd/gofermart/internal/models"
)

type Store interface {
	PopulateDB(ctx context.Context) error
	InitializeSecurityKey(ctx context.Context) error
	GetSecKey(ctx context.Context) (string, error)
	RegisterUser(ctx context.Context, reg *models.User) (int64, error)
	AuthUser(ctx context.Context, reg *models.User) (int64, error)
	CreateOrder(ctx context.Context, userid, orderid int64) error
	ListOrders(ctx context.Context, userid int64, orderList *[]models.Order) error
	GetBalance(ctx context.Context, userid int64, balance *models.Balance) error
	Withdraw(ctx context.Context, userid int64, withdraw *models.Withdraw) error
	Withdrawals(ctx context.Context, userid int64, withdrawals *[]models.Withdrawals) error
	AccrualSystemPoll(ctx context.Context, orders *[]int64) error
	AccrualSystemSave(ctx context.Context, accrual []models.Accrual) error
}
