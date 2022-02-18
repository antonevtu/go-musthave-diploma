package handlers

import (
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"net/http"
)

func balance(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func withdrawOrder(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func getWithdrawals(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
