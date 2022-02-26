package handlers

import (
	"encoding/json"
	"errors"
	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/antonevtu/go-musthave-diploma/internal/repository"
	"io"
	"log"
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

			err = repo.PostOrder(r.Context(), orderNum)
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
		orderList, err := repo.GetOrders(r.Context())
		if err != nil {
			log.Println("=====1", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(orderList) > 0 {
			//if len(orderList) == 2 {
			//	orderList = orderList[1:2]
			//}

			js, err := json.Marshal(orderList)
			if err != nil {
				log.Println("=====2", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Println("=====3", string(js))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err = w.Write(js)
			if err != nil {
				log.Println("=====4", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
