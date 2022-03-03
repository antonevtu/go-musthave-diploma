package app

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/antonevtu/go-musthave-diploma/internal/handlers"
	"github.com/antonevtu/go-musthave-diploma/internal/logger"
	"github.com/antonevtu/go-musthave-diploma/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"log"
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
	DatabaseURI = flag.String("d", "", "base url")
	AccrualSystemAddress = flag.String("r", "localhost:8080", "")
	SecretKey = flag.String("k", "SecretKey", "SecretKey")
	TokenPeriodExpire = flag.Int64("p", 500, "in hours")
	CtxTimeout = flag.Int64("t", 500, "context timeout")
}

func TestStatic(t *testing.T) {
	cfgApp := cfg.Config{
		RunAddress:           *RunAddress,
		DatabaseURI:          *DatabaseURI,
		AccrualSystemAddress: *AccrualSystemAddress,
		SecretKey:            *SecretKey,
		TokenPeriodExpire:    *TokenPeriodExpire,
		CtxTimeout:           *CtxTimeout,
	}

	zLog, err := logger.New(0)
	if err != nil {
		log.Fatal(err)
	}
	zLog.Infow("starting tests...")
	ctx := context.Background()

	// локальная БД
	db, err := repository.NewDB(context.Background(), *DatabaseURI, zLog, true)
	assert.Equal(t, err, nil)
	//err = db.DropTables(ctx)
	//require.NoError(t, err)
	//err = db.CreateTables(ctx)
	//require.NoError(t, err)

	// тестовый сервер
	r := handlers.NewRouter(&db, cfgApp)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// регистрация нового пользователя #1
	user := registerT{Login: uuid.NewString(), Password: uuid.NewString()}
	userJS1, _ := json.Marshal(user)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/user/register", bytes.NewBuffer(userJS1))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	cookiesTrue1 := resp.Cookies()
	_ = cookiesTrue1
	respBody, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
	require.NoError(t, err)
	resp.Body.Close()

	// повторная регистрация пользователя #1 (логин занят, 409)
	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/register", bytes.NewBuffer(userJS1))
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 409, resp.StatusCode)
	cookies := resp.Cookies()
	_ = cookies
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
	require.NoError(t, err)
	resp.Body.Close()

	// регистрация пользователя #2
	user = registerT{Login: uuid.NewString(), Password: uuid.NewString()}
	userJS2, _ := json.Marshal(user)

	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/register", bytes.NewBuffer(userJS2))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	cookiesTrue2 := resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
	require.NoError(t, err)
	resp.Body.Close()

	// аутентификация пользователя #1 успешная
	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/login", bytes.NewBuffer(userJS1))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	cookiesTrue1 = resp.Cookies()
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
	require.NoError(t, err)
	resp.Body.Close()

	// аутентификация пользователя не успешная (неверный пароль, 401)
	user1 := user
	user1.Password = "qwerty"
	userJS0, _ := json.Marshal(user1)

	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/login", bytes.NewBuffer(userJS0))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 401, resp.StatusCode)
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
	require.NoError(t, err)
	resp.Body.Close()

	// аутентификация пользователя не успешная (неверный логин, 401)
	user1 = user
	user1.Login = "qwerty"
	userJS0, _ = json.Marshal(user1)

	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/login", bytes.NewBuffer(userJS0))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 401, resp.StatusCode)
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
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
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
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
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
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
	//require.Equal(t, 409, resp.StatusCode)
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
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
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
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
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
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
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
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
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
	require.NoError(t, err)
	resp.Body.Close()

	// запрос на списание средств (200)
	err = db.PutTestAccrual(ctx)
	require.NoError(t, err)

	with := withdrawal{
		Order: "5404369895241180",
		Sum:   100,
	}
	withJS, _ := json.Marshal(with)

	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/balance/withdraw", bytes.NewBuffer(withJS))
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)
	req.AddCookie(cookiesTrue1[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
	require.NoError(t, err)
	resp.Body.Close()

	// запрос на списание средств (недостаточно средств 402)
	with = withdrawal{
		Order: "5404368922619749",
		Sum:   10000,
	}
	withJS, _ = json.Marshal(with)

	client = &http.Client{}
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/user/balance/withdraw", bytes.NewBuffer(withJS))
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)
	req.AddCookie(cookiesTrue1[0])
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 402, resp.StatusCode)
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
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
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
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
	respBody, err = ioutil.ReadAll(resp.Body)
	fmt.Println(string(respBody))
	require.NoError(t, err)
	resp.Body.Close()
}
