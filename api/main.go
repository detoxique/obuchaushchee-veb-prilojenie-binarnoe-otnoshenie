package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

const (
	secretKey = "XEf65D.dyLUkbWF0aDysauVm55n5fdADFpi/CuYNnOIOMSB8uQLZK" // Замените на свой секретный ключ
)

func main() {
	// Подключение к PostgreSQL
	connStr := "postgres://postgres:123@localhost/portaldb?sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("Database connection error:", err)
		return
	}
	defer db.Close()

	run()
}

type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Response struct {
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// Обработчик для api/auth
func handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginData LoginData
	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		sendError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Проверка пользователя в БД
	var storedHash string
	err = db.QueryRow("SELECT password FROM users WHERE username = $1", loginData.Username).Scan(&storedHash)
	if err == sql.ErrNoRows {
		sendError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		sendError(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Проверка пароля
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(loginData.Password))
	if err != nil {
		sendError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Создание JWT
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: loginData.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		sendError(w, "Could not generate token", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{Message: "Success", Token: tokenString}, http.StatusOK)
}

func sendResponse(w http.ResponseWriter, resp Response, status int) {
	fmt.Println("Response: " + resp.Message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func sendError(w http.ResponseWriter, message string, status int) {
	fmt.Println("Error: " + message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Message: message})
}

func verifyToken(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got verify request")
	tokenStr := r.Header.Get("Authorization")
	if tokenStr == "" {
		sendError(w, "No token provided", http.StatusUnauthorized)
		return
	}
	fmt.Println(tokenStr)

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil || !token.Valid {
		sendError(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	sendResponse(w, Response{Message: "Token valid"}, http.StatusOK)
}

func run() {
	http.HandleFunc("/api/auth", handleAuth)
	http.HandleFunc("/api/verify", verifyToken)

	fmt.Println("Auth server started at :1337")
	http.ListenAndServe(":1337", nil)
}
