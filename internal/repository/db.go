package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	"time"
)

const hashLen = 32 // SHA256

type DBT struct {
	*pgxpool.Pool
}

func NewDB(ctx context.Context, url string) (DBT, error) {
	var pool DBT
	var err error
	pool.Pool, err = pgxpool.Connect(ctx, url)
	if err != nil {
		return pool, err
	}

	// создание таблиц (см. create_tables.sql)
	sql := "create table if not exists users\n(\n    user_id serial primary key,\n    login varchar(64) unique,\n    pwd char(64),\n    pwd_salt char(64),\n    registered_at timestamp default now()\n);\n\ncreate table if not exists tokens\n(\n    id serial,\n    user_id integer,\n    key_salt char(64),\n    foreign key (user_id) references users (user_id) on delete cascade\n);\n\ncreate table if not exists orders\n(\n    id serial,\n    order_num varchar(32) primary key,\n    user_id integer,\n    uploaded_at timestamp default now(),\n    foreign key (user_id) references users (user_id) on delete cascade\n);\n\ncreate table if not exists accruals\n(\n    id serial ,\n    order_num varchar(32) primary key,\n    status varchar(10),\n    accrual numeric(12,2) default 0,\n    uploaded_at timestamp default now(),\n    foreign key (order_num) references orders (order_num) on delete cascade\n);\n\ncreate table if not exists withdrawns\n(\n    id serial,\n    order_num varchar(32) primary key,\n    withdrawn numeric(12,2),\n    processed_at timestamp default now(),\n    foreign key (order_num) references orders (order_num) on delete cascade\n);\n\ncreate table if not exists balance\n(\n    id serial primary key,\n    user_id integer unique,\n    available numeric(12,2) default 0 check (available >= 0),\n    withdrawn numeric(12,2) default 0 check (withdrawn >= 0),\n    foreign key (user_id) references users (user_id) on delete cascade\n);\n\ncreate table if not exists queue\n(\n    id serial primary key,\n    order_num varchar(32) unique,\n    user_id integer,\n    uploaded_at timestamp default now(),\n    last_checked_at timestamp default now(),\n    in_handling boolean default false\n);\n"
	_, err = pool.Exec(ctx, sql)
	if err != nil {
		return pool, err
	}

	return pool, nil
}

func (db *DBT) Register(ctx context.Context, login, password string, cfgApp cfg.Config) (token string, err error) {
	salt, err := RandBytes(hashLen)
	if err != nil {
		return "", err
	}

	pwdHash := ToHash(password, cfgApp.SecretKey, salt)

	tx, err := db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	// добавление пользователя в users
	sql := "insert into users (login, pwd, pwd_salt) values($1, $2, $3) returning user_id"
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

	// добавление баланса пользователя в balance
	sql1 := "insert into balance (user_id) values ($1);"
	_, err = db.Pool.Exec(ctx, sql1, userID)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("unable to commit: %w", err)
	}

	token, err = NewJwtToken(userID, cfgApp)
	if err != nil {
		return "", err
	}
	return token, err
}

func (db *DBT) Login(ctx context.Context, login, password string, cfgApp cfg.Config) (token string, err error) {
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
		return token, err
	} else {
		return "", ErrInvalidLoginPassword
	}
}

// TODO: сделать хранение ключей в БД
func (db *DBT) Authorize(ctx context.Context, token string, cfgApp cfg.Config) (userID int, err error) {
	userID, err = ParseToken(token, cfgApp.SecretKey)
	return userID, err
}

func (db *DBT) PostOrder(ctx context.Context, order string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// добавление заказа в orders. Проверка на уникальность
	userID := ctx.Value(UserIDKey).(int)
	sql := "insert into orders (order_num, user_id) values ($1, $2)"
	_, err = db.Pool.Exec(ctx, sql, order, userID)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {

		// конфликт номера заказов. Проверка, какой пользователь сделал заказ ранее
		if pgErr.Code == pgerrcode.UniqueViolation {
			sql1 := "select user_id from orders where order_num = $1;"
			resp := db.Pool.QueryRow(ctx, sql1, order)
			var userIDExist int
			err = resp.Scan(&userIDExist)
			if err != nil {
				return err
			}
			if userID == userIDExist {
				return ErrDuplicateOrderNumber
			} else {
				return ErrDuplicateOrderNumberByAnotherUser
			}
		}

	} else if err != nil {
		return err
	}

	// добавление номера заказов в историю и очередь на начисление баллов
	sql2 := "insert into accruals (order_num, status) values ($1, $2);"
	_, err = db.Pool.Exec(ctx, sql2, order, AccrualRegistered)
	if err != nil {
		return err
	}
	sql3 := "insert into queue (order_num, user_id) values ($1, $2);"
	_, err = db.Pool.Exec(ctx, sql3, order, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("unable to commit: %w", err)
	}
	return nil
}

func (db *DBT) GetOrders(ctx context.Context) (OrderList, error) {
	userID := ctx.Value(UserIDKey).(int)

	sql := "select order_num, status, accrual, uploaded_at from accruals where order_num in (select order_num from orders where user_id = $1);"
	rows, err := db.Pool.Query(ctx, sql, userID)
	if err != nil {
		return nil, err
	}

	res := make(OrderList, 0, 10)
	item := orderItem{}
	for rows.Next() {
		err = rows.Scan(&item.Number, &item.Status, &item.Accrual, &item.UploadedAtGo)
		if err != nil {
			return nil, err
		}
		item.UploadedAt = item.UploadedAtGo.Format(time.RFC3339)
		res = append(res, item)
	}
	return res, nil
}

func (db *DBT) Balance(ctx context.Context) (Balance, error) {
	userID := ctx.Value(UserIDKey).(int)
	sql := "select available, withdrawn from balance where user_id = $1"
	resp := db.Pool.QueryRow(ctx, sql, userID)
	bal := Balance{}
	err := resp.Scan(&bal.Current, &bal.Withdrawn)
	return bal, err
}

func (db *DBT) WithdrawToOrder(ctx context.Context, order string, sum float64) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// добавление заказа в orders. Проверка на уникальность
	userID := ctx.Value(UserIDKey).(int)
	sql := "insert into orders (order_num, user_id) values ($1, $2)"
	_, err = db.Pool.Exec(ctx, sql, order, userID)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// конфликт номера заказов. Проверка, какой пользователь сделал заказ ранее
		if pgErr.Code == pgerrcode.UniqueViolation {
			return ErrOrderAlreadyExists
		}
	}
	if err != nil {
		return err
	}

	// проверка баланса и списание
	sql2 := "update balance set available = available - $1, withdrawn = withdrawn + $1 where user_id = $2;"
	_, err = db.Pool.Exec(ctx, sql2, sum, userID)
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgerrcode.CheckViolation {
			return ErrNotEnoughFunds
		}
	}
	if err != nil {
		return err
	}

	// занесение в историю списаний
	sql1 := "insert into withdrawns (order_num, withdrawn) values ($1, $2);"
	_, err = db.Pool.Exec(ctx, sql1, order, sum)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("unable to commit: %w", err)
	}
	return err
}

func (db *DBT) GetWithdrawals(ctx context.Context) (WithdrawalsList, error) {
	userID := ctx.Value(UserIDKey).(int)

	sql := "select order_num, withdrawn, processed_at from withdrawns where order_num in (select order_num from orders where user_id = $1);"
	rows, err := db.Pool.Query(ctx, sql, userID)
	if err != nil {
		return nil, err
	}

	res := make(WithdrawalsList, 0, 10)
	item := withdrawalItem{}
	for rows.Next() {
		err = rows.Scan(&item.Order, &item.Sum, &item.ProcessedAtGo)
		if err != nil {
			return nil, err
		}
		item.ProcessedAt = item.ProcessedAtGo.Format(time.RFC3339)
		res = append(res, item)
	}
	return res, nil
}
