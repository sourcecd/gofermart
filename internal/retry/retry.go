package retry

import (
	"context"
	"errors"
	"time"

	baseretry "github.com/sethvargo/go-retry"
	"github.com/sourcecd/gofermart/internal/models"
	"github.com/sourcecd/gofermart/internal/prjerrors"
)

type (
	Retry struct {
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

func (retry *Retry) UserFuncRetry(f UserFunc) UserFunc {
	bf := baseretry.WithMaxRetries(retry.maxRetries, baseretry.NewFibonacci(retry.fiboDuration))

	return func(ctx context.Context, reg *models.User) (int64, error) {
		ctx, cancel := context.WithTimeout(ctx, retry.timeout)
		defer cancel()
		var userid int64
		var err error
		err = baseretry.Do(ctx, bf, func(ctx context.Context) error {
			userid, err = f(ctx, reg)
			if errors.Is(retry.skippedErrors, err) {
				return err
			}
			return baseretry.RetryableError(err)
		})
		return userid, err
	}
}

func (retry *Retry) CreateOrderFuncRetry(f CreateOrderFunc) CreateOrderFunc {
	bf := baseretry.WithMaxRetries(retry.maxRetries, baseretry.NewFibonacci(retry.fiboDuration))

	return func(ctx context.Context, userid, orderid int64) error {
		ctx, cancel := context.WithTimeout(ctx, retry.timeout)
		defer cancel()
		err := baseretry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, userid, orderid)
			if errors.Is(retry.skippedErrors, err) {
				return err
			}
			return baseretry.RetryableError(err)
		})
		return err
	}
}

func (retry *Retry) ListOrdersFuncRetry(f ListOrdersFunc) ListOrdersFunc {
	bf := baseretry.WithMaxRetries(retry.maxRetries, baseretry.NewFibonacci(retry.fiboDuration))

	return func(ctx context.Context, userid int64, orderList *[]models.Order) error {
		ctx, cancel := context.WithTimeout(ctx, retry.timeout)
		defer cancel()
		err := baseretry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, userid, orderList)
			if errors.Is(retry.skippedErrors, err) {
				return err
			}
			return baseretry.RetryableError(err)
		})
		return err
	}
}

func (retry *Retry) GetBalanceFuncRetry(f GetBalanceFunc) GetBalanceFunc {
	bf := baseretry.WithMaxRetries(retry.maxRetries, baseretry.NewFibonacci(retry.fiboDuration))

	return func(ctx context.Context, userid int64, balance *models.Balance) error {
		ctx, cancel := context.WithTimeout(ctx, retry.timeout)
		defer cancel()
		err := baseretry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, userid, balance)
			if errors.Is(retry.skippedErrors, err) {
				return err
			}
			return baseretry.RetryableError(err)
		})
		return err
	}
}

func (retry *Retry) WithdrawFuncRetry(f WithdrawFunc) WithdrawFunc {
	bf := baseretry.WithMaxRetries(retry.maxRetries, baseretry.NewFibonacci(retry.fiboDuration))

	return func(ctx context.Context, userid int64, withdraw *models.Withdraw) error {
		ctx, cancel := context.WithTimeout(ctx, retry.timeout)
		defer cancel()
		err := baseretry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, userid, withdraw)
			if errors.Is(retry.skippedErrors, err) {
				return err
			}
			return baseretry.RetryableError(err)
		})
		return err
	}
}

func (retry *Retry) WithdrawalsFuncRetry(f WithdrawalsFunc) WithdrawalsFunc {
	bf := baseretry.WithMaxRetries(retry.maxRetries, baseretry.NewFibonacci(retry.fiboDuration))

	return func(ctx context.Context, userid int64, withdrawals *[]models.Withdrawals) error {
		ctx, cancel := context.WithTimeout(ctx, retry.timeout)
		defer cancel()
		err := baseretry.Do(ctx, bf, func(ctx context.Context) error {
			err := f(ctx, userid, withdrawals)
			if errors.Is(retry.skippedErrors, err) {
				return err
			}
			return baseretry.RetryableError(err)
		})
		return err
	}
}

func (retry *Retry) SetParams(fibotime, timeout time.Duration, maxretries uint64) {
	retry.fiboDuration = fibotime
	retry.maxRetries = maxretries
	retry.timeout = timeout
}

func NewRetry() *Retry {
	return &Retry{
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

func (retry *Retry) GetTimeoutCtx() time.Duration {
	return retry.timeout
}
