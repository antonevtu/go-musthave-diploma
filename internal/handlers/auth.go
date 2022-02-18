package handlers

import (
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"net/http"
)

func register(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func login(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
