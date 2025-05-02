package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Секреты для токенов
var (
	accessTokenSecret  = []byte("jlesrvosreg")
	refreshTokenSecret = []byte("eriluvsekrg")
)

// CustomClaims структура с стандартными claims и пользовательскими данными
type CustomClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Генерация access-токена (действует 15 минут)
func GenerateAccessToken(username string) (string, error) {
	claims := CustomClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			Issuer:    "auth-service",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(accessTokenSecret)
}

// Генерация refresh-токена (действует 7 дней)
func GenerateRefreshToken(username string) (string, error) {
	claims := CustomClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			Issuer:    "auth-service",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(refreshTokenSecret)
}

func debugToken(tokenString string) {
	// Парсинг без проверки подписи
	token, _, _ := jwt.NewParser().ParseUnverified(tokenString, &CustomClaims{})
	if claims, ok := token.Claims.(*CustomClaims); ok {
		fmt.Printf("Token claims: %+v\n", claims)
		fmt.Printf("Signing alg: %v\n", token.Header["alg"])
		fmt.Printf("Time now: %v\n", time.Now())
	}
}

func CheckToken(tokenString string) (bool, error) {
	debugToken(tokenString)

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем, что алгоритм подписи тот же, что использовался при генерации
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return accessTokenSecret, nil
	})

	if err != nil || !token.Valid {
		return false, fmt.Errorf("invalid token: %v", err)
	}

	fmt.Println("Token validated correctly!")
	return true, nil
}
