package retr

import (
	"context"
	"errors"
	"time"

	"github.com/sethvargo/go-retry"
	"github.com/sourcecd/gofermart/internal/models"
	"github.com/sourcecd/gofermart/internal/prjerrors"
)

type (
	Retr struct {
		maxRetries uint64
		fiboDuration,
		timeout time.Duration
		skippedErrors error
	}

	UserFunc        func(ctx context.Context, reg *models.User) (int64, error)
	CreateOrderFunc func(ctx context.Context, userid, orderid int64) error
	ListOrdersFunc  func(ctx context.Context, userid int64, orderList *[]models.Order) error
	GetBalanceFunc  func(ctx context.Context, userid int64, balance *models.Balance) error
	WithdrawFunc    func(ctx context.Context, userid int64, withdraw *models.Withdraw) error
	WithdrawalsFunc func(ctx context.Context, userid int64, withdrawals *[]models.Withdrawals) error
)

func (rtr *Retr) UserFuncRetr(f UserFunc) UserFunc {
	bf := retry.WithMaxRetries(rtr.maxRetries, retry.NewFibonacci(rtr.fiboDuration))

	return func(ctx context.Context, reg *models.User) (int64, error) {
		ctx, cancel := context.WithTimeout(ctx, rtr.timeout)
		defer cancel()
		var userid int64
		var err error
		err = retry.Do(ctx, bf, func(ctx context.Context) error {
			userid, err = f(ctx, reg)
			if errors.Is(rtr.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return userid, err
	}
}

func (rtr *Retr) CreateOrderFuncRetr(f CreateOrderFunc) CreateOrderFunc {
	bf := retry.WithMaxRetries(rtr.maxRetries, retry.NewFibonacci(rtr.fiboDuration))

	return func(ctx context.Context, userid, orderid int64) error {
		ctx, cancel := context.WithTimeout(ctx, rtr.timeout)
		defer cancel()
		err := retry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, userid, orderid)
			if errors.Is(rtr.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return err
	}
}

func (rtr *Retr) ListOrdersFuncRetr(f ListOrdersFunc) ListOrdersFunc {
	bf := retry.WithMaxRetries(rtr.maxRetries, retry.NewFibonacci(rtr.fiboDuration))

	return func(ctx context.Context, userid int64, orderList *[]models.Order) error {
		ctx, cancel := context.WithTimeout(ctx, rtr.timeout)
		defer cancel()
		err := retry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, userid, orderList)
			if errors.Is(rtr.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return err
	}
}

func (rtr *Retr) GetBalanceFuncRetr(f GetBalanceFunc) GetBalanceFunc {
	bf := retry.WithMaxRetries(rtr.maxRetries, retry.NewFibonacci(rtr.fiboDuration))

	return func(ctx context.Context, userid int64, balance *models.Balance) error {
		ctx, cancel := context.WithTimeout(ctx, rtr.timeout)
		defer cancel()
		err := retry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, userid, balance)
			if errors.Is(rtr.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return err
	}
}

func (rtr *Retr) WithdrawFuncRetr(f WithdrawFunc) WithdrawFunc {
	bf := retry.WithMaxRetries(rtr.maxRetries, retry.NewFibonacci(rtr.fiboDuration))

	return func(ctx context.Context, userid int64, withdraw *models.Withdraw) error {
		ctx, cancel := context.WithTimeout(ctx, rtr.timeout)
		defer cancel()
		err := retry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, userid, withdraw)
			if errors.Is(rtr.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return err
	}
}

func (rtr *Retr) WithdrawalsFuncRetr(f WithdrawalsFunc) WithdrawalsFunc {
	bf := retry.WithMaxRetries(rtr.maxRetries, retry.NewFibonacci(rtr.fiboDuration))

	return func(ctx context.Context, userid int64, withdrawals *[]models.Withdrawals) error {
		ctx, cancel := context.WithTimeout(ctx, rtr.timeout)
		defer cancel()
		err := retry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, userid, withdrawals)
			if errors.Is(rtr.skippedErrors, err) {
				return err
			}
			return retry.RetryableError(err)
		})
		return err
	}
}

func (rtr *Retr) SetParams(fibotime, timeout time.Duration, maxretries uint64) {
	rtr.fiboDuration = fibotime
	rtr.maxRetries = maxretries
	rtr.timeout = timeout
}

func NewRetr() *Retr {
	return &Retr{
		fiboDuration: 1 * time.Second,
		maxRetries:   3,
		timeout:      30 * time.Second,
		skippedErrors: errors.Join(
			prjerrors.ErrAlreadyExists,
			prjerrors.ErrNotExists,
			prjerrors.ErrOrderAlreadyExists,
			prjerrors.ErrOtherOrderAlreadyExists,
			prjerrors.ErrEmptyData,
			prjerrors.ErrNotEnough,
		),
	}
}

func (rtr *Retr) GetTimeoutCtx() time.Duration {
	return rtr.timeout
}
