package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/antonevtu/go-musthave-diploma/internal/auth"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/antonevtu/go-musthave-diploma/internal/repository"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"
)

const minLoginLength = 4
const maxLoginLength = 64
const minPasswordLength = 4
const hashLen = 32 // SHA256

var secretKey = []byte("abc")

type registerT struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func register(repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
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

			// hashes
			pwdSalt, err := auth.RandBytes(hashLen)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			pwdHash := auth.ToHash(req.Password, cfgApp.SecretKey, pwdSalt)
			JWTSalt, err := auth.RandBytes(hashLen)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// register in repository
			reg := repository.RegisterNewUser{
				Login:   req.Login,
				PwdHash: pwdHash,
				PwdSalt: pwdSalt,
				JWTSalt: JWTSalt,
			}
			userID, err := repo.Register(r.Context(), reg)
			if errors.Is(err, repository.ErrLoginBusy) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// JWT-token
			token, err := auth.NewJwtToken(userID, cfgApp.SecretKey+JWTSalt, cfgApp.TokenPeriodExpire)
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
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
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
				http.Error(w, "invalid login or password", http.StatusUnauthorized)
				return
			}

			// find user in repository
			user, err := repo.Login(r.Context(), req.Login)
			if errors.Is(err, repository.ErrUnknownLogin) {
				http.Error(w, "invalid login or password", http.StatusUnauthorized)
				return
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// check user password
			pwdHash := auth.ToHash(req.Password, cfgApp.SecretKey, user.PwdSalt)
			if pwdHash != user.PwdHash {
				http.Error(w, "invalid login or password", http.StatusUnauthorized)
				return
			}

			// generate token
			JWTSalt, err := auth.RandBytes(hashLen)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			token, err := auth.NewJwtToken(user.UserID, cfgApp.SecretKey+JWTSalt, cfgApp.TokenPeriodExpire)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// save JWTSalt
			err = repo.UpdateTokenKey(r.Context(), user.UserID, JWTSalt)
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

type UserIDKeyT string

const (
	UserIDKey UserIDKeyT = "userID"
)

func middlewareAuth(next http.Handler, repo Repositorier, cfgApp cfg.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := extractToken(r)
		if err != nil {
			http.Error(w, "bad cookie", http.StatusUnauthorized)
			return
		}

		userID, err := auth.ExtractUserID(tokenString)
		if err != nil {
			http.Error(w, "bad cookie", http.StatusUnauthorized)
			return
		}

		salt, err := repo.GetTokenKey(r.Context(), userID)

		_, err = auth.ParseToken(tokenString, cfgApp.SecretKey+salt)

		if errors.Is(err, auth.ErrInvalidLoginPassword) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func correctLoginPassword(req registerT) bool {
	l := utf8.RuneCountInString(req.Login)
	p := utf8.RuneCountInString(req.Password)
	if (l >= minLoginLength) && (l <= maxLoginLength) && (p >= minPasswordLength) {
		return true
	}
	return false
}
