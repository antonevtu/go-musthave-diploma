package handlers

import (
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"net/http"
)

func postOrder(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func getOrders(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
