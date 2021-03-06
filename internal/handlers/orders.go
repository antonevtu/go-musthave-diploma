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

func postOrder(repo Repositorier, cfgApp cfg.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Type"), "text/plain") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()

			orderNum := string(body)

			err = goluhn.Validate(string(body))
			if err != nil {
				http.Error(w, "luhn validation failed", http.StatusUnprocessableEntity)
				return
			}

			userID := r.Context().Value(UserIDKey).(int)
			err = repo.PostOrder(r.Context(), userID, orderNum)
			if errors.Is(err, repository.ErrDuplicateOrderNumber) {
				w.WriteHeader(http.StatusOK)
				return
			}
			if errors.Is(err, repository.ErrDuplicateOrderNumberByAnotherUser) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusAccepted)

		} else {
			http.Error(w, "content-type is not text/plain", http.StatusBadRequest)
			return
		}
	})
}

func getOrders(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(UserIDKey).(int)
		orderList, err := repo.GetOrders(r.Context(), userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(orderList) > 0 {
			data, err := json.Marshal(orderList)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err = w.Write(data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
