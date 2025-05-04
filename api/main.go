package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
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

// Оценка
type Mark struct {
	Course    string    `json:"Course"`
	Date      time.Time `json:"Date"`
	MarkValue string    `json:"MarkValue"`
}

// Данные страницы профиля
type ProfilePageData struct {
	Username              string `json:"Username"`
	Role                  string `json:"Role"`
	Group                 string `json:"Group"`
	PerformancePercentage int    `json:"PerformancePersentage"`
	Marks                 []Mark `json:"Marks"`
}

// Авторизация
func handleAuth(w http.ResponseWriter, r *http.Request) {
	// Обрабатывать только POST запросы
	if r.Method != "POST" {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var loginData LoginData

	// Чтение JSON
	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		log.Println("Некорректный JSON")
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	// Проверка пользователя в БД
	var storedHash string
	err = db.QueryRow("SELECT password FROM users WHERE username = $1", loginData.Username).Scan(&storedHash)
	if err == sql.ErrNoRows {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Проверка пароля
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(loginData.Password))
	if err != nil {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	}

	// Генерация токенов
	accessToken, err := GenerateAccessToken(loginData.Username)
	if err != nil {
		log.Println("Не удалось создать access токен" + err.Error())
		http.Error(w, "Не удалось создать access токен", http.StatusInternalServerError)
		return
	}

	refreshToken, err := GenerateRefreshToken(loginData.Username)
	if err != nil {
		log.Println("Не удалось создать refresh токен")
		http.Error(w, "Не удалось создать refresh токен", http.StatusInternalServerError)
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
	log.Println("Получен запрос на верификацию токена")
	// Обрабатывать только POST запросы
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	token, err := extractToken(w, r)
	if err != nil {
		log.Println("Токен не валиден. " + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}

	tokenCheck, err := CheckAccessToken(token)
	if err != nil {
		log.Println("Токен не валиден. " + err.Error())
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	if !tokenCheck {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// Подтверждение токена админа
func verifyAdmin(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен запрос на верификацию токена админа")
	// Обрабатывать только POST запросы
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	token, err := extractToken(w, r)
	if err != nil {
		log.Println("Токен не валиден. " + err.Error())
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
	}

	tokenCheck, err := CheckAccessToken(token)
	if err != nil {
		log.Println("Токен не валиден. " + err.Error())
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	if !tokenCheck {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
	}

	username := GetUsernameGromToken(token)
	log.Println("Получаем данные для пользователя: " + username)

	// Получение данных из БД
	var role string
	err = db.QueryRow("SELECT role FROM users WHERE username = $1", username).Scan(&role)
	if err == sql.ErrNoRows {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if role == "admin" {
		log.Println("Пользователь является администратором")
		w.WriteHeader(http.StatusOK)
	} else {
		log.Println("Пользователь не является администратором")
		w.WriteHeader(http.StatusUnauthorized)
	}
}

// Получение access токена по refresh токену
func refreshToken(w http.ResponseWriter, r *http.Request) {
	// Обрабатывать только POST запросы
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	token, err := extractToken(w, r)
	if err != nil {
		log.Println("Токен не валиден. " + err.Error())
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
	}

	tokenCheck, err := CheckRefreshToken(token)
	if err != nil {
		log.Println("Токен не валиден. " + err.Error())
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	if !tokenCheck {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
	}

	// Если refresh токен валиден, генерируем новый access токен
	accessToken, err := GenerateAccessToken(GetUsernameGromToken(token))
	if err != nil {
		log.Println("Не удалось создать access токен. " + err.Error())
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
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

// Получение данных профиля по токену
func getProfileData(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен запрос на получение данных профиля")
	// Принимаются только POST запросы
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Вытаскиваем токен и получаем из него имя пользователя
	token, err := extractToken(w, r)
	if err != nil {
		log.Println("Ошибка при проверке токена. " + err.Error())
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
	}

	username := GetUsernameGromToken(token)
	log.Println("Получаем данные для пользователя: " + username)

	// Получение данных из БД
	var role, group string
	err = db.QueryRow("SELECT role FROM users WHERE username = $1", username).Scan(&role)
	if err == sql.ErrNoRows {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Роль пользователя " + username + ": " + role)

	err = db.QueryRow("SELECT groups.name FROM groups, users WHERE users.username = $1 AND users.id_group = groups.id", username).Scan(&group)
	if err == sql.ErrNoRows {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Группа пользователя " + username + ": " + group)

	data := ProfilePageData{
		Username:              username,
		Role:                  role,
		Group:                 group,
		PerformancePercentage: 100,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println("Ошибка в преобразовании в JSON")
		sendError(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}
	log.Println(string(jsonData))

	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

// Добавление пользователя через админ панель
func addUser(w http.ResponseWriter, r *http.Request) {

}

// Удаление пользователя через админ панель
func deleteUser(w http.ResponseWriter, r *http.Request) {

}

// Отправление сообщения об ошибке
func sendError(w http.ResponseWriter, message string, status int) {
	log.Println("Error: " + message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
}

func extractToken(w http.ResponseWriter, r *http.Request) (string, error) {
	// Вытаскиваем токен из запроса
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		//fmt.Printf("Error dumping request: %v\n", err)
		return "", err
	}
	fmt.Printf("Request Dump:\n%s\n", string(dump))

	token, err := extractBodyFromRequest(r)
	if err != nil {
		log.Println("Не удалось извлечь токен")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return "", err
	}

	token = removeCharAtIndex(token, 0)
	token = removeCharAtIndex(token, utf8.RuneCountInString(token)-1)

	return token, nil
}

// ExtractBodyFromRequest извлекает тело HTTP-запроса как строку
func extractBodyFromRequest(r *http.Request) (string, error) {
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
	http.HandleFunc("/api/verifyadmin", verifyAdmin)
	http.HandleFunc("/api/refreshtoken", refreshToken)
	http.HandleFunc("/api/getprofiledata", getProfileData)

	http.HandleFunc("/api/admin/adduser", addUser)
	http.HandleFunc("/api/admin/deleteuser", deleteUser)

	// Запуск сервера (Ctrl + C, чтобы выключить)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
