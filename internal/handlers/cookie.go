package handlers

import (
	"errors"
	"net/http"
)

const userIDCookieName = "user_auth"

func setCookie(w http.ResponseWriter, token string) {
	cook := http.Cookie{
		Name:  userIDCookieName,
		Value: token,
	}
	http.SetCookie(w, &cook)
}

func extractToken(r *http.Request) (token string, err error) {
	cook, errNoCookie := r.Cookie(userIDCookieName)
	if (cook != nil) && (errNoCookie == nil) {
		token = cook.Value
		return token, nil
	}
	return "", errors.New("no cookie")
}