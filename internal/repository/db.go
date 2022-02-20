package repository

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
)

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
