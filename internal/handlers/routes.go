package handlers

import (
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Repositorier interface {
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
		r.Post("/api/user/register", register(repo, cfgApp))                    // регистрация пользователя
		r.Post("/api/user/login", login(repo, cfgApp))                          // аутентификация пользователя
		r.Post("/api/user/orders", postOrder(repo, cfgApp))                     // загрузка пользователем номера заказа для расчета
		r.Get("/api/user/orders", getOrders(repo, cfgApp))                      // получение списка загруженных пользователем номеров звказов, статусов их обработки и информации о начислениях
		r.Get("/api/user/balance", balance(repo, cfgApp))                       // получение текущего баланса счета баллов лояльности пользователя
		r.Post("/api/user/balance/withdraw", withdrawOrder(repo, cfgApp))       // запрос на списание баллов с накопительного счета в счет оплаты нового заказа
		r.Delete("/api/user/balance/withdrawals", getWithdrawals(repo, cfgApp)) // получение информации о выводе средств с накопительног осчета пользователем
	})
	return r
}
