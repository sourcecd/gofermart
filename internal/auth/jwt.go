package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const tokenExp = 12 * time.Hour

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

func GenJWT(userID int, secKey string) (*string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		UserID: userID,
	})
	tokenString, err := token.SignedString([]byte(secKey))
	if err != nil {
		return nil, err
	}
	return &tokenString, nil
}

func ExtractJWT(tokenString string, secKey string) (*int, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secKey), nil
		})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("token is not valid")
	}

	return &claims.UserID, nil
}
