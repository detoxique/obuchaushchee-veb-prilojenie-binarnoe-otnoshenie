package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
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

type GroupData struct {
	GroupName string `json:"GroupName"`
}

type DeleteGroupData struct {
	Id string `json:"Id"`
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
	Username string `json:"Username"`
	Group    string `json:"Group"`
}

type TeacherCoursesPageData struct {
}

type CoursesPageData struct {
	Courses []string `json:"Courses"`
}

// Группа
type Group struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// Пользователь
type User struct {
	Id        int    `json:"id"`
	Username  string `json:"Username"`
	Role      string `json:"Role"`
	GroupName string `json:"GroupName"`
}

type UserData struct {
	Username  string `json:"Username"`
	Password  string `json:"Password"`
	Role      string `json:"Role"`
	GroupName string `json:"GroupName"`
}

// Данные на админ панели(группы и пользователи)
type AdminPanelData struct {
	Groups []Group `json:"Groups"`
	Users  []User  `json:"Users"`
}

type DeleteUserData struct {
	Name string `json:"Username"`
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

// Подтверждение токена препода
func verifyTeacher(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен запрос на верификацию токена препода")
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
	if role == "teacher" {
		log.Println("Пользователь является преподавателем")
		w.WriteHeader(http.StatusOK)
	} else {
		log.Println("Пользователь не является преподавателем")
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
		Username: username,
		Group:    group,
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

// Получение данных профиля по токену
func getTeacherProfileData(w http.ResponseWriter, r *http.Request) {
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

	if role != "teacher" {
		log.Println("Пользователь не является преподавателем")
		sendError(w, "Пользователь не является преподавателем ", http.StatusUnauthorized)
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
		Username: username,
		Group:    group,
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

func getTeacherCoursesData(w http.ResponseWriter, r *http.Request) {
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

	if role != "teacher" {
		log.Println("Пользователь не является преподавателем")
		sendError(w, "Пользователь не является преподавателем ", http.StatusUnauthorized)
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

	var data TeacherCoursesPageData

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

func getCoursesData(w http.ResponseWriter, r *http.Request) {
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

	if role != "student" {
		log.Println("Пользователь не является студентом")
		sendError(w, "Пользователь не является студентом ", http.StatusUnauthorized)
		return
	}

	log.Println("Роль пользователя " + username + ": " + role)

	var data CoursesPageData

	// Получение количества курсов в БД
	var coursesCount int
	err = db.QueryRow("SELECT COUNT(*) FROM courses").Scan(&coursesCount)
	if err == sql.ErrNoRows {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Количество курсов: " + strconv.Itoa(coursesCount))

	// Считывание курсов из БД
	courses := make([]string, coursesCount)
	for i := 1; i <= coursesCount; i++ {
		var id int
		var name string
		err = db.QueryRow("SELECT id, name FROM (SELECT *, ROW_NUMBER() OVER () as row_num FROM courses) AS subquery WHERE row_num = $1", i).Scan(&id, &name)
		if err == sql.ErrNoRows {
			log.Println("Неправильные данные")
			sendError(w, "Неправильные данные", http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Println("Ошибка базы данных")
			sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println(name)
		courses[i-1] = name
	}

	data.Courses = courses

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

// Получение данных админ панели
func getAdminPanelData(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен запрос на получение данных админ панели")
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

	if role != "admin" {
		log.Println("Ошибка доступа")
		sendError(w, "Ошибка доступа", http.StatusUnauthorized)
		return
	}

	// Получение количества групп в БД
	var groupsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM groups").Scan(&groupsCount)
	if err == sql.ErrNoRows {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Получение количества пользователей в БД
	var usersCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&usersCount)
	if err == sql.ErrNoRows {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Количество пользователей: " + strconv.Itoa(usersCount))

	// Считывание групп из БД
	groups := make([]Group, groupsCount)
	for i := 1; i <= groupsCount; i++ {
		var id int
		var name string
		err = db.QueryRow("SELECT id, name FROM (SELECT *, ROW_NUMBER() OVER () as row_num FROM groups) AS subquery WHERE row_num = $1", i).Scan(&id, &name)
		if err == sql.ErrNoRows {
			log.Println("Неправильные данные")
			sendError(w, "Неправильные данные", http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Println("Ошибка базы данных")
			sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
			return
		}
		groups[i-1] = Group{Id: id, Name: name}
	}

	// Считывание пользователей из БД
	users := make([]User, usersCount)
	for i := 1; i <= usersCount; i++ {
		var id int
		var name string
		var role string
		var group_id int
		err = db.QueryRow("SELECT id, username, role, id_group FROM (SELECT *, ROW_NUMBER() OVER () as row_num FROM users) AS subquery WHERE row_num = $1", i).Scan(&id, &name, &role, &group_id)
		if err == sql.ErrNoRows {
			log.Println("Неправильные данные")
			sendError(w, "Неправильные данные", http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Println("Ошибка базы данных")
			sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
			return
		}
		var groupName string
		for i := 0; i < groupsCount; i++ {
			if groups[i].Id == group_id {
				groupName = groups[i].Name
			}
		}
		users[i-1] = User{Id: id, Username: name, Role: role, GroupName: groupName}
	}

	// log.Printf("Содержимое массива: ")
	// for i := 0; i < groupsCount; i++ {
	// 	h := groups[i]
	// 	log.Printf(strconv.Itoa(h.Id) + " " + h.Name + " ")
	// }

	adminData := AdminPanelData{
		Groups: groups,
		Users:  users,
	}

	jsonData, err := json.Marshal(adminData)
	if err != nil {
		log.Println("Ошибка в преобразовании в JSON")
		sendError(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}
	log.Println(string(jsonData))

	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(adminData)
}

// Добавление пользователя через админ панель
func addUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var userData UserData

	// Чтение JSON
	err := json.NewDecoder(r.Body).Decode(&userData)
	if err != nil {
		log.Println("Некорректный JSON")
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	log.Println(userData.GroupName)

	// Проверка пользователя в БД
	var checkuser string
	err = db.QueryRow("SELECT username FROM users WHERE username = $1", userData.Username).Scan(&checkuser)
	if err != nil {
		if err == sql.ErrNoRows {
			// Пользователя нет в базе, продолжаем
			log.Println("Проверка прошла успешно, пользователей с таким именем нет")
			// Ищем ID группы
			var id int
			err = db.QueryRow("SELECT id FROM groups WHERE name = $1", userData.GroupName).Scan(&id)
			if err != nil {
				log.Println("Неправильные данные")
				sendError(w, "Неправильные данные", http.StatusUnauthorized)
			}

			hash, _ := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)

			_, err := db.Exec("INSERT INTO users (username, password, role, id_group) VALUES ($1, $2, $3, $4)", userData.Username, string(hash), userData.Role, id)
			if err != nil {
				log.Println("Ошибка базы данных")
				sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Успешный ответ
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
		}
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

}

// Добавление группы через админ панель
func addGroup(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен запрос на добавление группы через админ панель")
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var groupData GroupData

	// Чтение JSON
	err := json.NewDecoder(r.Body).Decode(&groupData)
	if err != nil {
		log.Println("Некорректный JSON")
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	log.Println(groupData.GroupName)

	// Проверка группы в БД
	var checkgroup string
	err = db.QueryRow("SELECT name FROM groups WHERE name = $1", groupData.GroupName).Scan(&checkgroup)
	if err != nil {
		if err == sql.ErrNoRows {
			// Группы нет в базе, продолжаем
			_, err = db.Exec("INSERT INTO groups (name) VALUES ($1)", groupData.GroupName)
			if err != nil {
				log.Println("Ошибка базы данных")
				sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Успешный ответ
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			return
		}
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

}

// Удаление пользователя через админ панель
func deleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var user DeleteUserData

	// Чтение JSON
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		log.Println("Некорректный JSON")
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}
	log.Println("Удаляем пользователя " + user.Name)
	// Проверка пользователя в БД
	var checkUser string
	err = db.QueryRow("SELECT username FROM users WHERE username = $1", user.Name).Scan(&checkUser)
	if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	var delete string
	err = db.QueryRow("DELETE FROM users WHERE username = $1", user.Name).Scan(&delete)
	if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// Удаление группы через админ панель
func deleteGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var data DeleteGroupData

	// Чтение JSON
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Println("Некорректный JSON")
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	// Проверка группы в БД
	var checkGroup string
	err = db.QueryRow("SELECT name FROM groups WHERE id = $1", data.Id).Scan(&checkGroup)
	if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Группа есть в базе

	_, err = db.Exec("DELETE FROM groups WHERE id = $1", data.Id)
	if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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
	http.HandleFunc("/api/verifyteacher", verifyTeacher)
	http.HandleFunc("/api/refreshtoken", refreshToken)
	http.HandleFunc("/api/getprofiledata", getProfileData)
	http.HandleFunc("/api/getteacherprofiledata", getTeacherProfileData)
	http.HandleFunc("/api/getteachercoursesdata", getTeacherCoursesData)
	http.HandleFunc("/api/getcoursesdata", getCoursesData)

	http.HandleFunc("/api/admin/getadminpaneldata", getAdminPanelData)
	http.HandleFunc("/api/admin/adduser", addUser)
	http.HandleFunc("/api/admin/deleteuser", deleteUser)
	http.HandleFunc("/api/admin/addgroup", addGroup)
	http.HandleFunc("/api/admin/deletegroup", deleteGroup)

	// Запуск сервера (Ctrl + C, чтобы выключить)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
