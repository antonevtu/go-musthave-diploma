package app

//
//import (
//	"context"
//	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
//	"github.com/antonevtu/go-musthave-diploma/internal/handlers"
//	"github.com/antonevtu/go-musthave-diploma/internal/repository"
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//	"net/http/httptest"
//	"testing"
//)
//
//func TestAccrual(t *testing.T) {
//	cfgApp := cfg.Config{
//		RunAddress:           *RunAddress,
//		DatabaseURI:          *DatabaseURI,
//		AccrualSystemAddress: *AccrualSystemAddress,
//		SecretKey:            *SecretKey,
//		TokenPeriodExpire:    *TokenPeriodExpire,
//		CtxTimeout:           *CtxTimeout,
//	}
//	ctx := context.Background()
//
//	// локальная БД
//	dbPool, err := repository.NewDB(context.Background(), *DatabaseURI)
//	assert.Equal(t, err, nil)
//	sql1 := "drop table if exists users, tokens, orders, accruals, withdrawns, balance, queue cascade;\n\ncreate table if not exists users\n(\n    user_id serial primary key,\n    login varchar(64) unique,\n    pwd char(64),\n    pwd_salt char(64),\n    registered_at timestamp default now()\n);\n\ncreate table if not exists tokens\n(\n    id serial,\n    user_id integer,\n    key_salt char(64),\n    foreign key (user_id) references users (user_id) on delete cascade\n);\n\ncreate table if not exists orders\n(\n    id serial,\n    order_num varchar(32) primary key,\n    user_id integer,\n    uploaded_at timestamp default now(),\n    foreign key (user_id) references users (user_id) on delete cascade\n);\n\ncreate table if not exists accruals\n(\n    id serial ,\n    order_num varchar(32) primary key,\n    status varchar(10),\n    accrual numeric(12,2) default 0,\n    uploaded_at timestamp default now(),\n    foreign key (order_num) references orders (order_num) on delete cascade\n);\n\ncreate table if not exists withdrawns\n(\n    id serial,\n    order_num varchar(32) primary key,\n    withdrawn numeric(12,2),\n    processed_at timestamp default now(),\n    foreign key (order_num) references orders (order_num) on delete cascade\n);\n\ncreate table if not exists balance\n(\n    id serial primary key,\n    user_id integer unique,\n    available numeric(12,2) default 0 check (available >= 0),\n    withdrawn numeric(12,2) default 0 check (withdrawn >= 0),\n    foreign key (user_id) references users (user_id) on delete cascade\n);\n\ncreate table if not exists queue\n(\n    id serial primary key,\n    order_num varchar(32) unique,\n    user_id integer,\n    uploaded_at timestamp default now(),\n    last_checked_at timestamp default now(),\n    in_handling boolean default false\n);"
//	_, err = dbPool.Exec(ctx, sql1)
//	require.NoError(t, err)
//
//	// тестовый сервер
//	r := handlers.NewRouter(&dbPool, cfgApp)
//	ts := httptest.NewServer(r)
//	defer ts.Close()
//
//	// пул опроса внешнего сервиса
//
//}
