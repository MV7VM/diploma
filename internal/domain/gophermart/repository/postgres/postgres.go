package postgres

// Package postgres implements the pgx-based data-access layer.
// It is Fx-compatible (provides lifecycle hooks) and reflects Forest Fairy
// «UUID edition» schema after the April-2025 migration.

import (
	"context"
	"errors"
	"fmt"
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
			return 0, entities.ErrAlreadyInUse
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

const qUploadOrder = `
WITH check_order AS (
    SELECT user_id FROM gophermart.orders WHERE order_number = $1
),
insert_order AS (
    INSERT INTO gophermart.orders (order_number, user_id, status)
    SELECT $1, $2, 0
    WHERE NOT EXISTS (SELECT 1 FROM check_order)
    RETURNING order_number
)
SELECT 
    CASE 
        WHEN EXISTS (SELECT 1 FROM check_order WHERE user_id = $2) THEN 'own_order'
        WHEN EXISTS (SELECT 1 FROM check_order WHERE user_id != $2) THEN 'other_order'
        WHEN EXISTS (SELECT 1 FROM insert_order) THEN 'created'
        ELSE 'error'
    END as result`

func (r *Repository) UploadOrder(ctx context.Context, userID int, order string) error {
	var res string
	err := r.db.QueryRow(ctx, qUploadOrder, order, userID).Scan(&res)
	if err != nil {
		return fmt.Errorf("failed to upload order: %w", err)
	}

	switch res {
	case "own_order":
		return entities.ErrAlreadyInUse
	case "other_order":
		return entities.ErrPermissionDenied
	case "created":
		return nil
	default:
		return fmt.Errorf("unexpected result: %s", res)
	}
}

const qGetOrder = `
select 
    order_number, os.name as status, null as accrual, upload_time 
from 
    gophermart.orders 
  left join 
    gophermart.order_status os on orders.status = os.id
where user_id = $1`

func (r *Repository) GetOrders(ctx context.Context, userID int) ([]entities.Order, error) {
	rows, err := r.db.Query(ctx, qGetOrder, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders, err := pgx.CollectRows(rows, pgx.RowToStructByName[entities.Order])
	if err != nil {
		return nil, err
	}

	return orders, nil
}

const qUploadWithdraw = `
insert into gophermart.withdraw (order_number, sum, user_id) values ($1, $2, $3)`

func (r *Repository) UploadWithdraw(ctx context.Context, withdraw *entities.Withdraw, userID int) error {
	_, err := r.db.Exec(ctx, qUploadWithdraw, withdraw.OrderNumber, withdraw.Sum, userID)
	if err != nil {
		return err
	}

	return nil
}

const qGetWithdraw = `
select 
    order_number, sum, upload_time 
from 
    gophermart.withdraw 
-- where 
--     user_id = $1`

func (r *Repository) GetWithdraw(ctx context.Context, userID int) ([]entities.Withdraw, error) {
	rows, err := r.db.Query(ctx, qGetWithdraw) //, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	withdraw, err := pgx.CollectRows(rows, pgx.RowToStructByName[entities.Withdraw])
	if err != nil {
		return nil, err
	}

	return withdraw, nil
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

	_, err = execFunc(ctx, `
		CREATE TABLE IF NOT EXISTS gophermart.order_status (
			id int PRIMARY KEY,       
			name TEXT NOT NULL UNIQUE         
		)
	`)
	if err != nil {
		return err
	}
	_, err = execFunc(ctx, `
	with insert_statuses as (
		insert into gophermart.order_status (id, name) 
		values (0, 'NEW'),(1, 'PROCESSING'),(2, 'INVALID'),(3, 'PROCESSED') 
		on conflict (name) do nothing
		returning id
	)
	select count(*) as inserted_count from insert_statuses
	`)
	if err != nil {
		return err
	}

	_, err = execFunc(ctx, `
		CREATE TABLE IF NOT EXISTS gophermart.orders (
			order_number TEXT PRIMARY KEY, 
			user_id int references gophermart.users(id),
			status int references gophermart.order_status(id),
			accrual int,
			upload_time timestamptz default now()                
		)
	`)
	if err != nil {
		return err
	}

	_, err = execFunc(ctx, `
		CREATE TABLE IF NOT EXISTS gophermart.withdraw (
			order_number TEXT PRIMARY KEY, 
			user_id int references gophermart.users(id),
			sum int,
			upload_time timestamptz default now()                
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
