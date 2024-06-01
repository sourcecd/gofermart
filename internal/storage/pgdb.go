package storage

import (
	"context"
	"database/sql"
	"errors"

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
	createUserRec   = "INSERT INTO users (login, password) values ($1, $2) RETURNING id"

	getUserRec = "SELECT id, login, password FROM users WHERE login=$1"
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
