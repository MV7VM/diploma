package postgres

// Package postgres implements the pgx-based data-access layer.
// It is Fx-compatible (provides lifecycle hooks) and reflects Forest Fairy
// «UUID edition» schema after the April-2025 migration.

import (
	"context"
	"errors"
	"strings"

	"github.com/MV7VM/diploma/internal/config"
	"github.com/MV7VM/diploma/internal/domain/gophermart/entities"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const txKey entities.CtxKeyString = "tx"

// -----------------------------------------------------------------------------
// Pg-repository (Fx-ready)
// -----------------------------------------------------------------------------

type Repository struct {
	ctx context.Context
	cfg *config.PsqlConfig
	db  *pgxpool.Pool
}

// NewRepository returns a Repo instance ready to be plugged into an Fx graph.
func NewRepository(ctx context.Context, cfg *config.Model) (*Repository, error) {
	return &Repository{ctx: ctx, cfg: &cfg.Repo.PsqlConfig}, nil
}

// OnStart — Fx Lifecycle hook: opens a pgx connection-pool (with retries).
func (r *Repository) OnStart(ctx context.Context) (err error) {
	r.db, err = pgxpool.New(r.ctx, r.cfg.PsqlConnString)
	if err != nil {
		return err
	}

	if err = r.withTx(ctx, func(ctxTx context.Context) error {
		tx := ctxTx.Value(txKey).(pgx.Tx)
		return r.migrate(ctx, tx)
	}); err != nil {
		return err
	}

	return nil
}

// OnStop — Fx hook: closes pool.
func (r *Repository) OnStop(_ context.Context) error {
	if r.db != nil {
		r.db.Close()
	}
	return nil
}

const qRegisterUser = `
INSERT INTO gophermart.users 
    (login, password) 
values 
    ($1, $2) 
returning id`

func (r *Repository) RegisterUser(ctx context.Context, auth *entities.UserAuth) (id int, err error) {
	err = r.db.QueryRow(ctx, qRegisterUser, auth.Login, auth.Password).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "constraint") {
			return 0, entities.ErrLoginAlreadyInUse
		}

		return 0, err
	}

	return id, nil
}

const qLoginUser = `
select 
    id 
from 
    gophermart.users 
where 
    login = $1 
  and 
    password = $2`

func (r *Repository) LoginUser(ctx context.Context, auth *entities.UserAuth) (id int, err error) {
	err = r.db.QueryRow(ctx, qLoginUser, auth.Login, auth.Password).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, entities.ErrWrongCredentials
		}

		return 0, err
	}

	return id, nil
}

// migrate создает схему и таблицу для хранения URL, если они не существуют.
// Если tx == nil, операции выполняются напрямую через пул соединений.
func (r *Repository) migrate(ctx context.Context, tx pgx.Tx) error {
	var execFunc func(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	if tx != nil {
		execFunc = tx.Exec
	} else {
		execFunc = r.db.Exec
	}

	// Создаем схему shortener, если её нет
	_, err := execFunc(ctx, `CREATE SCHEMA IF NOT EXISTS gophermart`)
	if err != nil {
		return err
	}

	// Создаем таблицу urls, если её нет
	_, err = execFunc(ctx, `
		CREATE TABLE IF NOT EXISTS gophermart.users (
			id bigserial PRIMARY KEY, 
			login TEXT NOT NULL unique, 
			password TEXT
		)
	`)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) withTx(ctx context.Context, f func(context.Context) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	ctxTx := context.WithValue(ctx, txKey, tx)

	err = f(ctxTx)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
