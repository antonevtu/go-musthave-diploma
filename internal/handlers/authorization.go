package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/antonevtu/go-musthave-diploma/internal/repository"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"unicode/utf8"
)

const minLoginLength = 4
const maxLoginLength = 32
const minPasswordLength = 4

var secretKey = []byte("abc")

type registerT struct {
	login string
	password string
}


func register(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			http.Error(w, fmt.Errorf("invalid length login or password: %s", err).Error(), http.StatusBadRequest)
			return
		}

		// register in repository
		token, err := repo.Register(req.login, req.password)
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
	}
}

func login(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			http.Error(w, fmt.Errorf("invalid length login or password: %s", err).Error(), http.StatusBadRequest)
			return
		}

		// authentication in repository
		token, err := repo.Login(req.login, req.password)
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
	}
}

func middlewareAuth(next http.Handler, repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := extractToken(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		userID, err := repo.Authorize(tokenString)
		if errors.Is(err, repository.ErrInvalidLoginPassword) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		next.ServeHTTP(w, r)
	}
}

/*
func userAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := extractToken(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		authorized := repo.
		//if r.Header.Get(`Content-Encoding`) == `gzip` {
		//	gz, err := gzip.NewReader(r.Body)
		//	if err != nil {
		//		http.Error(w, err.Error(), http.StatusInternalServerError)
		//		return
		//	}
		//	defer gz.Close()
		//	r.Body = gz
		//	next.ServeHTTP(w, r)
		//} else {
		//	next.ServeHTTP(w, r)
		//}
	})
}
*/

func correctLoginPassword(req registerT) bool {
	l := utf8.RuneCountInString(req.login)
	p := utf8.RuneCountInString(req.password)
	if (l >= minLoginLength) && (l <= maxLoginLength) && (p >= minPasswordLength) {
		return true
	}
	return false
}