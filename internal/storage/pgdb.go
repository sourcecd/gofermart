package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/sourcecd/gofermart/internal/crypto"
	"github.com/sourcecd/gofermart/internal/models"
	"github.com/sourcecd/gofermart/internal/prjerrors"
)

type PgDB struct {
	db *sql.DB
}

//go:embed migrations/*.sql
var embedMigrations embed.FS

const (
	checkSecurityKey = "SELECT COUNT (id) FROM security"
	createSecureKey  = "INSERT INTO security (seckey) VALUES ($1)"
	getSecurityKey   = "SELECT seckey FROM security"

	createUserRec = "INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id"

	getUserRec = "SELECT id, login, password FROM users WHERE login=$1"

	createOrderRec = "INSERT INTO orders (userid, number, uploaded_at, processable, processed, status) VALUES ($1, $2, $3, $4, $5, 'NEW')"
	checkOrderRec  = "SELECT userid FROM orders WHERE number=$1"

	listOrders = "SELECT number, uploaded_at, status, accrual FROM orders WHERE (userid=$1 AND processable=true) ORDER BY uploaded_at DESC"

	checkBalance = "SELECT current, withdrawn FROM balance WHERE userid=$1"

	withdrawOp             = "UPDATE balance SET current=(current - $1), withdrawn=(withdrawn + $1) WHERE userid=$2"
	createOrderRecWithdraw = "INSERT INTO orders (userid, number, sum, processed_at, processable) VALUES ($1, $2, $3, $4, $5)"

	getWithdrawals = "SELECT number, sum, processed_at FROM orders WHERE (userid=$1 AND processable=false) ORDER BY processed_at DESC"

	accrualPollReq = "SELECT number FROM orders WHERE (processable=true AND processed=false)"
	accrualUpdate  = "UPDATE orders SET status=$1, accrual=$2, processed=$3 WHERE number=$4"
	accrualBalance = "INSERT INTO balance (userid, current, withdrawn) VALUES ($2, $1, 0) ON CONFLICT (userid) DO UPDATE SET current=(balance.current + $1)"
)

func NewDB(dsn string) (*PgDB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &PgDB{
		db: db,
	}, nil
}

func (pg *PgDB) CreateDatabaseScheme(ctx context.Context) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.UpContext(ctx, pg.db, "migrations"); err != nil {
		return err
	}

	return nil
}

func (pg *PgDB) InitializeSecurityKey(ctx context.Context) error {
	var count int64
	row := pg.db.QueryRowContext(ctx, checkSecurityKey)
	if err := row.Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		seckey, err := crypto.GenerateRandomKey()
		if err != nil {
			return err
		}
		if _, err = pg.db.ExecContext(ctx, createSecureKey, seckey); err != nil {
			return err
		}
	}

	return nil
}

func (pg *PgDB) GetSecurityKey(ctx context.Context) (string, error) {
	var seckey string
	row := pg.db.QueryRowContext(ctx, getSecurityKey)
	if err := row.Scan(&seckey); err != nil {
		return "", err
	}

	return seckey, nil
}

func (pg *PgDB) RegisterUser(ctx context.Context, reg *models.User) (int64, error) {
	var id int64
	err := pg.db.QueryRowContext(ctx, createUserRec, reg.Login, crypto.GeneratePasswordHash(reg.Password)).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			return -1, prjerrors.ErrAlreadyExists
		}
		return -1, err
	}
	return id, nil
}

func (pg *PgDB) AuthUser(ctx context.Context, reg *models.User) (int64, error) {
	var (
		id int64
		login,
		password string
	)
	row := pg.db.QueryRowContext(ctx, getUserRec, reg.Login)
	if err := row.Scan(&id, &login, &password); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, prjerrors.ErrNotExists
		}
		return -1, err
	}
	if crypto.GeneratePasswordHash(reg.Password) == password {
		return id, nil
	}
	return -1, prjerrors.ErrNotExists
}

func (pg *PgDB) CreateOrder(ctx context.Context, userid, orderid int64) error {
	var checkUserID int64
	if _, err := pg.db.ExecContext(ctx, createOrderRec, userid, orderid, time.Now(), true, false); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			row := pg.db.QueryRowContext(ctx, checkOrderRec, orderid)
			if err := row.Scan(&checkUserID); err != nil {
				return err
			}
			if checkUserID == userid {
				return prjerrors.ErrOrderAlreadyExists
			}
			return prjerrors.ErrOtherOrderAlreadyExists
		}
	}
	return nil
}

func (pg *PgDB) ListOrders(ctx context.Context, userid int64, orderList *[]models.Order) error {
	var (
		number     int64
		uploadedAt time.Time
		status     string
		accrual    sql.NullFloat64

		rowsCount int64
	)
	rows, err := pg.db.QueryContext(ctx, listOrders, userid)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&number, &uploadedAt, &status, &accrual); err != nil {
			return err
		}
		*orderList = append(*orderList, models.Order{
			Number:     fmt.Sprint(number),
			UploadedAt: uploadedAt.Format(time.RFC3339),
			Status:     status,
			Accrual:    accrual.Float64,
		})
		rowsCount++
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	if rowsCount == 0 {
		return prjerrors.ErrEmptyData
	}
	return nil
}

func (pg *PgDB) GetBalance(ctx context.Context, userid int64, balance *models.Balance) error {
	var (
		current   float64
		withdrawn float64
	)
	row := pg.db.QueryRowContext(ctx, checkBalance, userid)
	if err := row.Scan(&current, &withdrawn); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			//may be need to create default balance rec
			return nil
		}
		return err
	}
	balance.Current = current
	balance.Withdrawn = withdrawn
	return nil
}

func (pg *PgDB) Withdraw(ctx context.Context, userid int64, withdraw *models.Withdraw) error {
	tx, err := pg.db.Begin()
	defer tx.Rollback()
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, withdrawOp, withdraw.Sum, userid)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			return prjerrors.ErrNotEnough
		}
		return err
	}
	r, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if r == 0 {
		return prjerrors.ErrNotEnough
	}
	num, err := strconv.Atoi(withdraw.Order)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, createOrderRecWithdraw, userid, num, withdraw.Sum, time.Now(), false); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			return prjerrors.ErrOrderAlreadyExists
		}
		return err
	}
	return tx.Commit()
}

func (pg *PgDB) Withdrawals(ctx context.Context, userid int64, withdrawals *[]models.Withdrawals) error {
	var (
		number      int64
		sum         float64
		processedAt time.Time

		rowsCount int64
	)
	rows, err := pg.db.QueryContext(ctx, getWithdrawals, userid)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&number, &sum, &processedAt); err != nil {
			return err
		}
		*withdrawals = append(*withdrawals, models.Withdrawals{
			Order:       fmt.Sprint(number),
			Sum:         sum,
			ProcessedAt: processedAt.Format(time.RFC3339),
		})
		rowsCount++
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	if rowsCount == 0 {
		return prjerrors.ErrEmptyData
	}
	return nil
}

func (pg *PgDB) AccrualSystemPoll(ctx context.Context, orders *[]int64) error {
	var number int64
	rows, err := pg.db.QueryContext(ctx, accrualPollReq)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&number); err != nil {
			return err
		}
		*orders = append(*orders, number)
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	return nil
}

func (pg *PgDB) AccrualSystemSave(ctx context.Context, accrual []models.Accrual) error {
	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, v := range accrual {
		num, err := strconv.Atoi(v.Order)
		if err != nil {
			slog.Error(err.Error())
		}
		switch v.Status {
		case "PROCESSED":
			var userid int64
			if _, err := tx.ExecContext(ctx, accrualUpdate, v.Status, v.Accrual, true, num); err != nil {
				return err
			}
			if err := pg.db.QueryRowContext(ctx, checkOrderRec, num).Scan(&userid); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, accrualBalance, v.Accrual, userid); err != nil {
				return err
			}
		case "PROCESSING":
			if _, err := tx.ExecContext(ctx, accrualUpdate, v.Status, v.Accrual, false, num); err != nil {
				return err
			}
		case "INVALID":
			if _, err := tx.ExecContext(ctx, accrualUpdate, v.Status, v.Accrual, true, num); err != nil {
				return err
			}
		case "REGISTERED":
			if _, err := tx.ExecContext(ctx, accrualUpdate, v.Status, v.Accrual, false, num); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
