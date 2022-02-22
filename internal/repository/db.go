package repository

import (
	"context"
	"errors"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
)

const hashLen = 32 // SHA256

type dbT struct {
	*pgxpool.Pool
}

func NewDB(ctx context.Context, url string) (dbT, error) {
	var pool dbT
	var err error
	pool.Pool, err = pgxpool.Connect(ctx, url)
	if err != nil {
		return pool, err
	}

	// создание таблиц (см. create_tables.sql)
	sql := "create table if not exists users\n(\n    user_id serial primary key,\n    login varchar(32) unique,\n    password_hash varchar,\n    registered_at timestamp default now()\n    );\n\ncreate table if not exists orders\n(\n    id serial,\n    order_num integer primary key,\n    user_id integer,\n    uploaded_at timestamp default now(),\n    foreign key (user_id) references users (user_id) on delete cascade\n    );\n\ncreate table if not exists accruals\n(\n    id serial ,\n    order_num integer primary key,\n    status varchar(10),\n    accrual integer,\n    uploaded_at timestamp default now(),\n    foreign key (order_num) references orders (order_num) on delete cascade\n    );\n\ncreate table if not exists withdrawals\n(\n    id serial,\n    order_num integer primary key,\n    withdrawal integer,\n    uploaded_at timestamp default now(),\n    foreign key (order_num) references orders (order_num) on delete cascade\n    );\n\ncreate table if not exists balance\n(\n    id serial primary key,\n    user_id integer,\n    available integer,\n    withdrawals integer,\n    foreign key (user_id) references users (user_id) on delete cascade\n    );\n\ncreate table if not exists cache\n(\n    id serial primary key,\n    user_id integer,\n    order_id integer unique,\n    uploaded_at timestamp default now(),\n    last_checked_at timestamp default now()\n);"
	_, err = pool.Exec(ctx, sql)
	if err != nil {
		return pool, err
	}

	return pool, nil
}

func (db *dbT) Register(ctx context.Context, login, password string, cfgApp cfg.Config) (token string, err error) {
	salt, err := RandBytes(hashLen)
	if err != nil {
		return "", err
	}

	pwdHash := ToHash(password, cfgApp.SecretKey, salt)

	sql := "insert into users values(default, '$1', '$2', '$3') returning user_id"
	resp := db.Pool.QueryRow(ctx, sql, login, pwdHash, salt)

	var userID int
	var pgErr *pgconn.PgError

	err = resp.Scan(&userID)
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgerrcode.UniqueViolation {
			return "", ErrLoginBusy
		}
	} else if err != nil {
		return "", err
	}

	token, err = NewJwtToken(userID, cfgApp)
	if err != nil {
		return "", err
	}
	return token, err
}

func (db *dbT) Login(ctx context.Context, login, password string, cfgApp cfg.Config) (token string, err error) {
	sql := "select user_id, pwd, pwd_salt from users where login = $1"
	resp := db.Pool.QueryRow(ctx, sql, login)

	var userID int
	var pwdBase, salt string
	err = resp.Scan(&userID, &pwdBase, &salt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidLoginPassword
	}
	if err != nil {
		return "", err
	}

	pwdHash := ToHash(password, cfgApp.SecretKey, salt)
	if pwdHash == pwdBase {
		token, err = NewJwtToken(userID, cfgApp)
		if err != nil {
			return "", err
		}
	}
	return "", ErrInvalidLoginPassword
}

func (db *dbT) Authorize(ctx context.Context, token string) (userID int, err error) {
	userID := ctx.Value("userID").(int)

}

func (db *dbT) PostOrder(ctx context.Context, order int) error {

}

func (db *dbT) GetOrders(ctx context.Context) (OrdersList, error) {

}

func (db *dbT) Balance(ctx context.Context) (Balance, error) {

}

func (db *dbT) WithdrawToOrder(ctx context.Context, order int, sum float64) error {

}

func (db *dbT) GetWithdrawals(ctx context.Context) (WithdrawalsList, error) {

}
