package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/dgrijalva/jwt-go/v4"
	"time"
)

type jwtClaims struct {
	jwt.StandardClaims
	UserID int `json:"user_id"`
}

var ErrInvalidLoginPassword = errors.New("invalid login/password pair")

func NewJwtToken(userID int, secretKey string, tokenPeriodExpire int64) (string, error) {
	cl := jwtClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: jwt.At(time.Now().Add(time.Duration(tokenPeriodExpire) * time.Hour)),
			IssuedAt:  jwt.At(time.Now()),
		},
		UserID: userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &cl)

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		panic(err)
	}
	return tokenString, err
}

func ToHash(s string, key, salt string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(s))
	h.Write([]byte(salt))
	dst := h.Sum(nil)
	res := hex.EncodeToString(dst)
	return res
}

func RandBytes(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return ``, err
	}
	return hex.EncodeToString(b), nil
}

func ParseToken(accessToken string, signingKey string) (int, error) {

	claims := new(jwtClaims)
	token, err := jwt.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(signingKey), nil
	})
	if err != nil {
		return 0, err
	}
	if claims1, ok := token.Claims.(*jwtClaims); ok && token.Valid {
		return claims1.UserID, nil
	} else {
		return 0, ErrInvalidLoginPassword
	}
}
