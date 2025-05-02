package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"unicode/utf8"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Порт
const port string = ":1337"

// БД
var db *sql.DB

// Данные для входа
type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Ответ сервера
type Response struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Токен
type TokenResponse struct {
	AccessToken string `json:"Authorization"`
}

// Авторизация
func handleAuth(w http.ResponseWriter, r *http.Request) {
	// Обрабатывать только POST запросы
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginData LoginData

	// Чтение JSON
	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		log.Println("Invalid request")
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Проверка пользователя в БД
	var storedHash string
	err = db.QueryRow("SELECT password FROM users WHERE username = $1", loginData.Username).Scan(&storedHash)
	if err == sql.ErrNoRows {
		log.Println("Invalid credentials")
		sendError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Database error")
		sendError(w, "Database error "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Проверка пароля
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(loginData.Password))
	if err != nil {
		log.Println("Invalid credentials")
		sendError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Генерация токенов
	accessToken, err := GenerateAccessToken(loginData.Username)
	if err != nil {
		log.Println("Failed to generate access token" + err.Error())
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := GenerateRefreshToken(loginData.Username)
	if err != nil {
		log.Println("Failed to generate refresh token")
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := Response{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Подтверждение токена
func verifyToken(w http.ResponseWriter, r *http.Request) {
	log.Println("Got verify request")
	// Обрабатывать только POST запросы
	if r.Method != "POST" {
		log.Println("Method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}
	fmt.Printf("Request Dump:\n%s\n", string(dump))

	token, err := ExtractBodyFromRequest(r)
	if err != nil {
		log.Println("Extract failed")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	token = removeCharAtIndex(token, 0)
	token = removeCharAtIndex(token, utf8.RuneCountInString(token)-1)

	tokenCheck, err := CheckAccessToken(token)
	if err != nil {
		log.Println("Token validate failed. " + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if !tokenCheck {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// Получение access токена по refresh токену
func refreshToken(w http.ResponseWriter, r *http.Request) {
	// Обрабатывать только POST запросы
	if r.Method != "POST" {
		log.Println("Method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Вытаскиваем токен из запроса
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}
	fmt.Printf("Request Dump:\n%s\n", string(dump))

	token, err := ExtractBodyFromRequest(r)
	if err != nil {
		log.Println("Extract failed")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	token = removeCharAtIndex(token, 0)
	token = removeCharAtIndex(token, utf8.RuneCountInString(token)-1)

	tokenCheck, err := CheckRefreshToken(token)
	if err != nil {
		log.Println("Token validate failed. " + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if !tokenCheck {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
	}

	// Если refresh токен валиден, генерируем новый access токен
	accessToken, err := GenerateAccessToken(GetUsernameGromToken(token))
	if err != nil {
		log.Println("Failed to generate token. " + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := Response{
		AccessToken:  accessToken,
		RefreshToken: token,
	}

	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func sendError(w http.ResponseWriter, message string, status int) {
	log.Println("Error: " + message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
}

// ExtractBodyFromRequest извлекает тело HTTP-запроса как строку
func ExtractBodyFromRequest(r *http.Request) (string, error) {
	// Читаем тело запроса
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	// Закрываем тело запроса
	defer r.Body.Close()

	// Преобразуем байты в строку
	bodyString := string(bodyBytes)
	return bodyString, nil
}

func removeCharAtIndex(s string, index int) string {
	if index < 0 || index >= len(s) {
		return ""
	}
	return s[:index] + s[index+1:]
}

func connectDB() {
	// Подключение к PostgreSQL
	connStr := "postgres://postgres:123@localhost/portaldb?sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Println("Database connection error:", err)
		return
	}
	//defer db.Close()
}

func main() {
	connectDB()

	log.Println("Сервер API запущен на " + port)

	http.HandleFunc("/api/auth", handleAuth)
	http.HandleFunc("/api/verify", verifyToken)
	http.HandleFunc("/api/refreshtoken", refreshToken)

	// Запуск сервера (Ctrl + C, чтобы выключить)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
