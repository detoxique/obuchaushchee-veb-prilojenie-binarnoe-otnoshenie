package main

import (
	"time"

	"github.com/dgrijalva/jwt-go"
)

// Секреты для токенов
const (
	accessTokenSecret  = "jkl89Grh8G"
	refreshTokenSecret = "nlgRGeirg7"
)

// Генерация access-токена (действует 15 минут)
func GenerateAccessToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(accessTokenSecret))
}

// Генерация refresh-токена (действует 7 дней)
func GenerateRefreshToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(refreshTokenSecret))
}
