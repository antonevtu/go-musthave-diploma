package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/dgrijalva/jwt-go/v4"
	"time"
)

type jwtClaims struct {
	jwt.StandardClaims
	UserID int `json:"user_id"`
}

func newJwtToken(userId int, cfgApp cfg.Config) (string, error) {
	cl := jwtClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: jwt.At(time.Now().Add(time.Duration(cfgApp.TokenPeriodExpire) * time.Hour)),
			IssuedAt: jwt.At(time.Now()),
		},
		UserID: userId,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &cl)

	tokenString, err := token.SignedString(cfgApp.)
	if err != nil {
		panic(err)
	}
	return tokenString, err
}

func toHash(s string, key, salt string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(s))
	h.Write([]byte(salt))
	dst := h.Sum(nil)
	res := hex.EncodeToString(dst)
	return res
}

func randBytes(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return ``, err
	}
	return hex.EncodeToString(b), nil
}

func parseToken(accessToken string, signingKey []byte) (string, error) {
	token, err := jwt.ParseWithClaims(accessToken, )
}