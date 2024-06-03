package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sourcecd/gofermart/internal/cryptandsign"
	"github.com/sourcecd/gofermart/internal/models"
	"github.com/sourcecd/gofermart/internal/prjerrors"
)

type PgDB struct {
	db *sql.DB
}

var (
	createSecureTable = "CREATE TABLE IF NOT EXISTS security (id BIGSERIAL PRIMARY KEY, seckey VARCHAR(255))"
	checkSecurityKey  = "SELECT COUNT (id) FROM security"
	createSecureKey   = "INSERT INTO security (seckey) VALUES ($1)"
	getSecurityKey    = "SELECT seckey FROM security"

	createUserTable = "CREATE TABLE IF NOT EXISTS users (id BIGSERIAL, login VARCHAR(255) PRIMARY KEY, password VARCHAR(255))"
	createUserRec   = "INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id"

	getUserRec = "SELECT id, login, password FROM users WHERE login=$1"

	createOrdersTable = "CREATE TABLE IF NOT EXISTS orders (userid BIGINT, number BIGINT PRIMARY KEY, uploaded_at TIMESTAMPTZ)"
	createOrderRec    = "INSERT INTO orders (userid, number, uploaded_at) VALUES ($1, $2, $3)"
	checkOrderRec     = "SELECT userid FROM orders WHERE number=$1"

	listOrders = "SELECT number, uploaded_at FROM orders WHERE userid=$1 ORDER BY uploaded_at DESC"
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

func (pg *PgDB) PopulateDB(ctx context.Context) error {
	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, createSecureTable); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, createUserTable); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, createOrdersTable); err != nil {
		return err
	}

	return tx.Commit()
}

func (pg *PgDB) InitSecKey(ctx context.Context) error {
	var count int
	row := pg.db.QueryRowContext(ctx, checkSecurityKey)
	if err := row.Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		seckey, err := cryptandsign.GenRandKey()
		if err != nil {
			return err
		}
		if _, err = pg.db.ExecContext(ctx, createSecureKey, *seckey); err != nil {
			return err
		}
	}

	return nil
}

func (pg *PgDB) GetSecKey(ctx context.Context) (*string, error) {
	var seckey string
	row := pg.db.QueryRowContext(ctx, getSecurityKey)
	if err := row.Scan(&seckey); err != nil {
		return nil, err
	}

	return &seckey, nil
}

func (pg *PgDB) RegisterUser(ctx context.Context, reg *models.User) (*int, error) {
	var id int
	err := pg.db.QueryRowContext(ctx, createUserRec, reg.Login, cryptandsign.GetPassHash(reg.Password)).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			return nil, prjerrors.ErrAlreadyExists
		}
		return nil, err
	}
	return &id, nil
}

func (pg *PgDB) AuthUser(ctx context.Context, reg *models.User) (*int, error) {
	var (
		id int
		login,
		password string
	)
	row := pg.db.QueryRowContext(ctx, getUserRec, reg.Login)
	if err := row.Scan(&id, &login, &password); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, prjerrors.ErrNotExists
		}
		return nil, err
	}
	if cryptandsign.GetPassHash(reg.Password) == password {
		return &id, nil
	}
	return nil, prjerrors.ErrNotExists
}

func (pg *PgDB) CreateOrder(ctx context.Context, userid, orderid int) error {
	var checkUserID int
	if _, err := pg.db.ExecContext(ctx, createOrderRec, userid, orderid, time.Now()); err != nil {
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

func (pg *PgDB) ListOrders(ctx context.Context, userid int, orderList *[]models.Order) error {
	var (
		number     int
		uploadedAt time.Time

		rowsCount int
	)
	rows, err := pg.db.QueryContext(ctx, listOrders, userid)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&number, &uploadedAt); err != nil {
			return err
		}
		*orderList = append(*orderList, models.Order{
			Number:     number,
			UploadedAt: uploadedAt.Format(time.RFC3339),
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
