package main

import (
	"fmt"
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

func CheckToken(tokenString string) bool {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(accessTokenSecret), nil
	})
	if err != nil {
		fmt.Println("JWT check error " + err.Error())
		return false
	}

	// Проверяем валидность токена
	if !token.Valid {
		return false
	}

	return true
}
