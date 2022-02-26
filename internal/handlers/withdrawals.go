package handlers

import (
	"encoding/json"
	"errors"
	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/antonevtu/go-musthave-diploma/internal/repository"
	"io"
	"net/http"
	"strings"
)

func getBalance(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bal, err := repo.Balance(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		js, err := json.Marshal(bal)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(js)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type withdrawal struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func withdrawToOrder(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()

			req := withdrawal{}
			err = json.Unmarshal(body, &req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			err = goluhn.Validate(req.Order)
			if err != nil {
				http.Error(w, "luhn validation failed", http.StatusUnprocessableEntity)
			}

			err = repo.WithdrawToOrder(r.Context(), req.Order, req.Sum)
			if errors.Is(err, repository.ErrNotEnoughFunds) {
				http.Error(w, err.Error(), http.StatusPaymentRequired)
			}
			if errors.Is(err, repository.ErrOrderAlreadyExists) {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			w.WriteHeader(http.StatusOK)
		} else {
			http.Error(w, "invalid content-type: must be application/json", http.StatusBadRequest)
		}
	}
}

func getWithdrawals(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wl, err := repo.GetWithdrawals(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(wl) > 0 {
			js, err := json.Marshal(wl)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err = w.Write(js)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
