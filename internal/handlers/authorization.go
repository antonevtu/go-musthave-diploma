package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/antonevtu/go-musthave-diploma/internal/repository"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"
)

const minLoginLength = 4
const maxLoginLength = 32
const minPasswordLength = 4

var secretKey = []byte("abc")

type registerT struct {
	login    string
	password string
}

func register(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()

			req := registerT{}
			err = json.Unmarshal(body, &req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// check login, password for length, special symbols
			if !correctLoginPassword(req) {
				http.Error(w, "invalid login or password length", http.StatusBadRequest)
				return
			}

			// register in repository
			token, err := repo.Register(r.Context(), req.login, req.password)
			if errors.Is(err, repository.ErrLoginBusy) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			setCookie(w, token)
			w.WriteHeader(http.StatusOK)

		} else {
			http.Error(w, "invalid content-type: must be application/json", http.StatusBadRequest)
			return
		}
	}
}

func login(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()

			req := registerT{}
			err = json.Unmarshal(body, &req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// check login, password for length, TODO: special symbols
			if !correctLoginPassword(req) {
				http.Error(w, "invalid login or password length", http.StatusBadRequest)
				return
			}

			// authentication in repository
			token, err := repo.Login(r.Context(), req.login, req.password)
			if errors.Is(err, repository.ErrInvalidLoginPassword) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			setCookie(w, token)
			w.WriteHeader(http.StatusOK)

		} else {
			http.Error(w, "invalid content-type: must be application/json", http.StatusBadRequest)
			return
		}

	}
}

func middlewareAuth(next http.Handler, repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := extractToken(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		userID, err := repo.Authorize(r.Context(), tokenString)
		if errors.Is(err, repository.ErrInvalidLoginPassword) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func correctLoginPassword(req registerT) bool {
	l := utf8.RuneCountInString(req.login)
	p := utf8.RuneCountInString(req.password)
	if (l >= minLoginLength) && (l <= maxLoginLength) && (p >= minPasswordLength) {
		return true
	}
	return false
}
