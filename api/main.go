package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

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
}

// Обработчик для api/login
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

	sendResponse(w, Response{Message: "Success"}, http.StatusOK)
}

func sendResponse(w http.ResponseWriter, resp Response, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Message: message})
}

func run() {
	http.HandleFunc("/api/auth", handleAuth)
	fmt.Println("Auth server started at :1337")
	http.ListenAndServe(":1337", nil)
}
