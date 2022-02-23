package handlers

import (
	"context"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/antonevtu/go-musthave-diploma/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Repositorier interface {
	Register(ctx context.Context, login string, password string, cfgApp cfg.Config) (token string, err error)
	Login(ctx context.Context, login string, password string, cfgApp cfg.Config) (token string, err error)
	Authorize(ctx context.Context, token string, cfgApp cfg.Config) (userID int, err error)
	PostOrder(ctx context.Context, order int) error
	GetOrders(ctx context.Context) (repository.OrderList, error)
	Balance(ctx context.Context) (repository.Balance, error)
	WithdrawToOrder(ctx context.Context, order int, sum float64) error
	GetWithdrawals(ctx context.Context) (repository.WithdrawalsList, error)
}

func NewRouter(repo Repositorier, cfgApp cfg.Config) chi.Router {
	// Определяем роутер chi
	r := chi.NewRouter()

	// зададим встроенные middleware, чтобы улучшить стабильность приложения
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// архивирование запроса/ответа gzip
	r.Use(gzipResponseHandle)
	r.Use(gzipRequestHandle)

	// создадим суброутер
	r.Route("/", func(r chi.Router) {
		r.Post("/api/user/register", register(repo, cfgApp))                                                  // регистрация пользователя
		r.Post("/api/user/login", login(repo, cfgApp))                                                        // аутентификация пользователя
		r.Post("/api/user/orders", middlewareAuth(postOrder(repo, cfgApp), repo, cfgApp))                     // загрузка пользователем номера заказа для расчета
		r.Get("/api/user/orders", middlewareAuth(getOrders(repo, cfgApp), repo, cfgApp))                      // получение списка загруженных пользователем номеров звказов, статусов их обработки и информации о начислениях
		r.Get("/api/user/balance", middlewareAuth(getBalance(repo, cfgApp), repo, cfgApp))                    // получение текущего баланса счета баллов лояльности пользователя
		r.Post("/api/user/balance/withdraw", middlewareAuth(withdrawToOrder(repo, cfgApp), repo, cfgApp))     // запрос на списание баллов с накопительного счета в счет оплаты нового заказа
		r.Delete("/api/user/balance/withdrawals", middlewareAuth(getWithdrawals(repo, cfgApp), repo, cfgApp)) // получение информации о выводе средств с накопительног осчета пользователем
	})
	return r
}
