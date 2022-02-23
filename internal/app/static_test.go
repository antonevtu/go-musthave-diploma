package app

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/antonevtu/go-musthave-diploma/internal/handlers"
	"github.com/antonevtu/go-musthave-diploma/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type registerT struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type withdrawal struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

var (
	RunAddress           *string
	DatabaseURI          *string
	AccrualSystemAddress *string
	SecretKey            *string
	TokenPeriodExpire    *int64
	CtxTimeout           *int64
)

func init() {
	RunAddress = flag.String("a", ":8081", "server address for shorten")
	DatabaseURI = flag.String("d", "postgres://postgres:5069@localhost:5432/postgres", "base url")
	AccrualSystemAddress = flag.String("r", "http://localhost:8080", "")
	SecretKey = flag.String("k", "SecretKey", "SecretKey")
	TokenPeriodExpire = flag.Int64("p", 500, "in hours")
	CtxTimeout = flag.Int64("t", 500, "context timeout")
}

func TestIntegration(t *testing.T) {
	cfgApp := cfg.Config{
		RunAddress:           *RunAddress,
		DatabaseURI:          *DatabaseURI,
		AccrualSystemAddress: *AccrualSystemAddress,
		SecretKey:            *SecretKey,
		TokenPeriodExpire:    *TokenPeriodExpire,
		CtxTimeout:           *CtxTimeout,
	}
	ctx := context.Background()

	// локальная БД
	dbPool, err := repository.NewDB(context.Background(), *DatabaseURI)
	assert.Equal(t, err, nil)
	sql1 := "drop table if exists users, tokens, orders, accruals, withdrawns, balance, queue cascade;\n\ncreate table if not exists users\n(\n    user_id serial primary key,\n    login varchar(64) unique,\n    pwd char(64),\n    pwd_salt char(64),\n    registered_at timestamp default now()\n);\n\ncreate table if not exists tokens\n(\n    id serial,\n    user_id integer,\n    key_salt char(64),\n    foreign key (user_id) references users (user_id) on delete cascade\n);\n\ncreate table if not exists orders\n(\n    id serial,\n    order_num varchar(32) primary key,\n    user_id integer,\n    uploaded_at timestamp default now(),\n    foreign key (user_id) references users (user_id) on delete cascade\n);\n\ncreate table if not exists accruals\n(\n    id serial ,\n    order_num varchar(32) primary key,\n    status varchar(10),\n    accrual numeric(12,2) default 0,\n    uploaded_at timestamp default now(),\n    foreign key (order_num) references orders (order_num) on delete cascade\n);\n\ncreate table if not exists withdrawns\n(\n    id serial,\n    order_num varchar(32) primary key,\n    withdrawn numeric(12,2),\n    processed_at timestamp default now(),\n    foreign key (order_num) references orders (order_num) on delete cascade\n);\n\ncreate table if not exists balance\n(\n    id serial primary key,\n    user_id integer unique,\n    available numeric(12,2) default 0 check (available >= 0),\n    withdrawn numeric(12,2) default 0 check (withdrawn >= 0),\n    foreign key (user_id) references users (user_id) on delete cascade\n);\n\ncreate table if not exists queue\n(\n    id serial primary key,\n    order_num varchar(32) unique,\n    user_id integer,\n    uploaded_at timestamp default now(),\n    last_checked_at timestamp default now()\n);"
	_, err = dbPool.Exec(ctx, sql1)
	require.NoError(t, err)

	// тестовый сервер
	r := handlers.NewRouter(&dbPool, cfgApp)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// регистрация нового пользователя
	user := registerT{Login: uuid.NewString(), Password: uuid.NewString()}
	js, _ := json.Marshal(user)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/user/register", bytes.NewBuffer(js))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	cookiesTrue1 := resp.Cookies()
	_ = cookiesTrue1
	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	_ = respBody

	// повторная регистрация пользователя (логин занят, 409)
	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/register", bytes.NewBuffer(js))
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 409, resp.StatusCode)
	cookies := resp.Cookies()
	_ = cookies
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// регистрация второго пользователя
	user = registerT{Login: uuid.NewString(), Password: uuid.NewString()}
	js, _ = json.Marshal(user)

	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/register", bytes.NewBuffer(js))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	cookiesTrue2 := resp.Cookies()
	_ = cookiesTrue2
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	_ = respBody

	// аутентификация пользователя успешная
	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/login", bytes.NewBuffer(js))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// аутентификация пользователя не успешная (неверный пароль, 401)
	user1 := user
	user1.Password = "qwerty"
	js, _ = json.Marshal(user1)

	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/login", bytes.NewBuffer(js))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 401, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// аутентификация пользователя не успешная (неверный логин, 401)
	user1 = user
	user1.Login = "qwerty"
	js, _ = json.Marshal(user1)

	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/login", bytes.NewBuffer(js))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 401, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// загрузка нового номера заказа
	order := []byte("5404361084409447")

	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/orders", bytes.NewBuffer(order))
	req.Header.Set("Content-Type", "text/plain")
	require.NoError(t, err)
	req.AddCookie(cookiesTrue1[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 202, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// загрузка повторно номера заказа тем же пользователем (200)
	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/orders", bytes.NewBuffer(order))
	req.Header.Set("Content-Type", "text/plain")
	require.NoError(t, err)
	req.AddCookie(cookiesTrue1[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// загрузка повторно номера заказа другим пользователем (409)
	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/orders", bytes.NewBuffer(order))
	req.Header.Set("Content-Type", "text/plain")
	require.NoError(t, err)
	req.AddCookie(cookiesTrue2[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 409, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// загрузка второго нового номера заказа
	order2 := []byte("5404361051028451")

	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/orders", bytes.NewBuffer(order2))
	req.Header.Set("Content-Type", "text/plain")
	require.NoError(t, err)
	req.AddCookie(cookiesTrue1[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 202, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// получение списка загруженных номеров заказов пользователя 1 (200)
	client = &http.Client{}
	req, err = http.NewRequest(http.MethodGet, ts.URL+"/api/user/orders", bytes.NewBuffer([]byte("")))
	require.NoError(t, err)
	req.AddCookie(cookiesTrue1[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// получение списка загруженных номеров заказов пользователя 2 (204 - нет контента)
	client = &http.Client{}
	req, err = http.NewRequest(http.MethodGet, ts.URL+"/api/user/orders", bytes.NewBuffer([]byte("")))
	require.NoError(t, err)
	req.AddCookie(cookiesTrue2[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 204, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// получение текущего баланса пользователя (200)
	client = &http.Client{}
	req, err = http.NewRequest(http.MethodGet, ts.URL+"/api/user/balance", bytes.NewBuffer([]byte("")))
	require.NoError(t, err)
	req.AddCookie(cookiesTrue1[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// запрос на списание средств (200)
	sql2 := "update balance set available = 1000;"
	_, err = dbPool.Exec(ctx, sql2)
	require.NoError(t, err)

	with := withdrawal{
		Order: "5404369895241180",
		Sum:   100,
	}
	js, _ = json.Marshal(with)

	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/balance/withdraw", bytes.NewBuffer(js))
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)
	req.AddCookie(cookiesTrue1[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// запрос на списание средств (недостаточно средств 402)
	with = withdrawal{
		Order: "5404368922619749",
		Sum:   10000,
	}
	js, _ = json.Marshal(with)

	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/balance/withdraw", bytes.NewBuffer(js))
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)
	req.AddCookie(cookiesTrue1[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 402, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// получение информации о выводе средств (есть списания - 200)
	client = &http.Client{}
	req, err = http.NewRequest(http.MethodGet, ts.URL+"/api/user/withdrawals", bytes.NewBuffer([]byte("")))
	require.NoError(t, err)
	req.AddCookie(cookiesTrue1[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// получение информации о выводе средств (нет списаний - 204)
	client = &http.Client{}
	req, err = http.NewRequest(http.MethodGet, ts.URL+"/api/user/withdrawals", bytes.NewBuffer([]byte("")))
	require.NoError(t, err)
	req.AddCookie(cookiesTrue2[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 204, resp.StatusCode)
	cookies = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
}
