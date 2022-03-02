package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/antonevtu/go-musthave-diploma/internal/logger"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"time"
)

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
	sql := "create table if not exists users\n(\n    user_id serial primary key,\n    login varchar(64) unique,\n    pwd char(64),\n    pwd_salt char(64),\n    registered_at timestamp default now()\n);\n\ncreate table if not exists tokens\n(\n    id serial,\n    user_id integer,\n    key_salt char(64),\n    foreign key (user_id) references users (user_id) on delete cascade\n);\n\ncreate table if not exists orders\n(\n    id serial,\n    order_num varchar(32) primary key,\n    user_id integer,\n    uploaded_at timestamp default now(),\n    foreign key (user_id) references users (user_id) on delete cascade\n);\n\ncreate table if not exists accruals\n(\n    id serial ,\n    order_num varchar(32) primary key,\n    status varchar(16),\n    accrual numeric(12,2) default 0,\n    uploaded_at timestamp default now(),\n    foreign key (order_num) references orders (order_num) on delete cascade\n);\n\ncreate table if not exists withdrawns\n(\n    id serial,\n    order_num varchar(32) primary key,\n    withdrawn numeric(12,2),\n    processed_at timestamp default now(),\n    foreign key (order_num) references orders (order_num) on delete cascade\n);\n\ncreate table if not exists balance\n(\n    id serial primary key,\n    user_id integer unique,\n    available numeric(12,2) default 0 check (available >= 0),\n    withdrawn numeric(12,2) default 0 check (withdrawn >= 0),\n    foreign key (user_id) references users (user_id) on delete cascade\n);\n\ncreate table if not exists queue\n(\n    id serial primary key,\n    order_num varchar(32) unique,\n    user_id integer,\n    uploaded_at timestamp default now(),\n    last_checked_at timestamp default now(),\n    in_handling boolean default false\n);"
	_, err = pool.Exec(ctx, sql)
	if err != nil {
		return pool, err
	}

	return pool, nil
}

func (db *DBT) Register(ctx context.Context, user RegisterNewUser) (userID int, err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	// добавление пользователя в users
	sql := "insert into users (login, pwd, pwd_salt) values($1, $2, $3) returning user_id"
	resp := db.Pool.QueryRow(ctx, sql, user.Login, user.PwdHash, user.PwdSalt)

	var pgErr *pgconn.PgError

	err = resp.Scan(&userID)
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgerrcode.UniqueViolation {
			return 0, ErrLoginBusy
		}
	} else if err != nil {
		return 0, err
	}

	// добавление баланса пользователя в balance
	sql1 := "insert into balance (user_id) values ($1);"
	_, err = db.Pool.Exec(ctx, sql1, userID)
	if err != nil {
		return 0, err
	}

	// добавление ключа jwt-токена в tokens
	sql2 := "insert into tokens (user_id, key_salt) values ($1, $2);"
	_, err = db.Pool.Exec(ctx, sql2, userID, user.JWTSalt)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("unable to commit: %w", err)
	}

	return userID, err
}

func (db *DBT) Login(ctx context.Context, login string) (user LoginUser, err error) {
	sql := "select user_id, pwd, pwd_salt from users where login = $1"
	resp := db.Pool.QueryRow(ctx, sql, login)
	err = resp.Scan(&user.UserID, &user.PwdHash, &user.PwdSalt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user, ErrUnknownLogin
	}
	if err != nil {
		return user, err
	}

	return user, nil
}

func (db *DBT) UpdateTokenKey(ctx context.Context, userID int, key string) (err error) {
	sql := "update tokens set key_salt = $1 where user_id = $2;"
	_, err = db.Pool.Exec(ctx, sql, key, userID)
	return err
}

func (db *DBT) GetTokenKey(ctx context.Context, userID int) (key string, err error) {
	sql := "select key_salt from tokens where user_id = $1;"
	resp := db.Pool.QueryRow(ctx, sql, userID)
	err = resp.Scan(&key)
	return key, err
}

func (db *DBT) PostOrder(ctx context.Context, userID int, order string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// добавление заказа в orders. Проверка на уникальность
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
	_, err = db.Pool.Exec(ctx, sql2, order, AccrualNew)
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

	zLog := ctx.Value(logger.Z).(*zap.SugaredLogger)
	zLog.Debugw("Принят заказ:", order, "userID:", userID)
	return nil
}

func (db *DBT) GetOrders(ctx context.Context, userID int) (OrderList, error) {

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

func (db *DBT) Balance(ctx context.Context, userID int) (Balance, error) {
	sql := "select available, withdrawn from balance where user_id = $1"
	resp := db.Pool.QueryRow(ctx, sql, userID)
	bal := Balance{}
	err := resp.Scan(&bal.Current, &bal.Withdrawn)
	return bal, err
}

func (db *DBT) WithdrawToOrder(ctx context.Context, userID int, order string, sum float64) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// добавление заказа в orders. Проверка на уникальность
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

func (db *DBT) GetWithdrawals(ctx context.Context, userID int) (WithdrawalsList, error) {

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
