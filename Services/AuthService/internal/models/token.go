package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TokenExpirationTime = time.Hour

var secretKey = []byte("whatthehellisgoingon")

type Claims struct {
	Email  string
	UserId int
	jwt.RegisteredClaims
}

func GenerateToken(email string, userId int) (time.Time, string, error) {
	expirationTime := time.Now().Add(TokenExpirationTime)
	claims := Claims{
		Email:  email,
		UserId: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	result, err := token.SignedString(secretKey)
	return expirationTime, result, err
}

func ValidateToken(tokenString string) (bool, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secretKey, nil
	})

	if err != nil {
		return false, err
	}

	_, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return false, jwt.ErrTokenMalformed
	}

	return true, nil
}

func GetClaimsFromToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenMalformed
	}

	return claims, nil
}
