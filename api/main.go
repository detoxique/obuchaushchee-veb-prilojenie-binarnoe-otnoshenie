package main

import (
	"api/internal/handler"
	"api/internal/models"
	"api/internal/repository"
	"api/internal/service"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gorilla/mux"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Порт
const port string = ":1337"

// БД
var Db *sql.DB

// Авторизация
func handleAuth(w http.ResponseWriter, r *http.Request) {
	// Обрабатывать только POST запросы
	if r.Method != "POST" {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var loginData models.LoginData

	// Чтение JSON
	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		log.Println("Некорректный JSON")
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	// Проверка пользователя в БД
	var storedHash string
	err = Db.QueryRow("SELECT password FROM users WHERE username = $1", loginData.Username).Scan(&storedHash)
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
	accessToken, err := service.GenerateAccessToken(loginData.Username)
	if err != nil {
		log.Println("Не удалось создать access токен" + err.Error())
		http.Error(w, "Не удалось создать access токен", http.StatusInternalServerError)
		return
	}

	refreshToken, err := service.GenerateRefreshToken(loginData.Username)
	if err != nil {
		log.Println("Не удалось создать refresh токен")
		http.Error(w, "Не удалось создать refresh токен", http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := models.Response{
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

	tokenCheck, err := service.CheckAccessToken(token)
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

	tokenCheck, err := service.CheckAccessToken(token)
	if err != nil {
		log.Println("Токен не валиден. " + err.Error())
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	if !tokenCheck {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
	}

	username := service.GetUsernameGromToken(token)
	log.Println("Получаем данные для пользователя: " + username)

	// Получение данных из БД
	var role string
	err = Db.QueryRow("SELECT role FROM users WHERE username = $1", username).Scan(&role)
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

	tokenCheck, err := service.CheckAccessToken(token)
	if err != nil {
		log.Println("Токен не валиден. " + err.Error())
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	if !tokenCheck {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
	}

	username := service.GetUsernameGromToken(token)
	log.Println("Получаем данные для пользователя: " + username)

	// Получение данных из БД
	var role string
	err = Db.QueryRow("SELECT role FROM users WHERE username = $1", username).Scan(&role)
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

	tokenCheck, err := service.CheckRefreshToken(token)
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
	accessToken, err := service.GenerateAccessToken(service.GetUsernameGromToken(token))
	if err != nil {
		log.Println("Не удалось создать access токен. " + err.Error())
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := models.Response{
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

	username := service.GetUsernameGromToken(token)
	log.Println("Получаем данные для пользователя: " + username)

	// Получение данных из БД
	var role, group string
	err = Db.QueryRow("SELECT role FROM users WHERE username = $1", username).Scan(&role)
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

	err = Db.QueryRow("SELECT groups.name FROM groups, users WHERE users.username = $1 AND users.id_group = groups.id", username).Scan(&group)
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

	data := models.ProfilePageData{
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

	username := service.GetUsernameGromToken(token)
	log.Println("Получаем данные для пользователя: " + username)

	// Получение данных из БД
	var role, group string
	err = Db.QueryRow("SELECT role FROM users WHERE username = $1", username).Scan(&role)
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

	err = Db.QueryRow("SELECT groups.name FROM groups, users WHERE users.username = $1 AND users.id_group = groups.id", username).Scan(&group)
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

	data := models.ProfilePageData{
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

	username := service.GetUsernameGromToken(token)
	log.Println("Получаем данные для пользователя: " + username)

	// Получение данных из БД
	var role, group string
	err = Db.QueryRow("SELECT role FROM users WHERE username = $1", username).Scan(&role)
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

	err = Db.QueryRow("SELECT groups.name FROM groups, users WHERE users.username = $1 AND users.id_group = groups.id", username).Scan(&group)
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

	var data models.TeacherCoursesPageData

	// Получение количества курсов в БД
	var coursesCount int
	err = Db.QueryRow(`SELECT COUNT(uc.id)
FROM users u
JOIN users_courses uc ON u.id = uc.id_user
JOIN courses c ON uc.id_course = c.id
WHERE u.username = $1`, username).Scan(&coursesCount)
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
	courses := make([]models.Course, coursesCount)
	for i := 1; i <= coursesCount; i++ {
		var id int
		var name string
		err = Db.QueryRow(`WITH numbered_rows AS (
    	SELECT c.id, c.name, ROW_NUMBER() OVER (ORDER BY c.id DESC) as row_num
	FROM users u
	JOIN users_courses uc ON u.id = uc.id_user
	JOIN courses c ON uc.id_course = c.id
	WHERE u.username = $1
)
SELECT id, name
FROM numbered_rows
WHERE row_num = $2`, username, i).Scan(&id, &name)
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
		courses[i-1] = models.Course{Id: id, Name: name}
	}

	data.Courses = courses

	// Считывание групп из БД
	var groupsCount int
	err = Db.QueryRow("SELECT COUNT(*) FROM groups").Scan(&groupsCount)
	if err == sql.ErrNoRows {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Количество групп " + strconv.Itoa(groupsCount))

	groups := make([]models.Group, groupsCount)
	for i := 1; i <= groupsCount; i++ {
		var id int
		var name string
		err = Db.QueryRow("SELECT id, name FROM (SELECT *, ROW_NUMBER() OVER () as row_num FROM groups) AS subquery WHERE row_num = $1", i).Scan(&id, &name)
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
		groups[i-1] = models.Group{Id: id, Name: name}
	}

	data.Groups = groups

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

	username := service.GetUsernameGromToken(token)
	log.Println("Получаем данные для пользователя: " + username)

	// Получение данных из БД
	var role string
	err = Db.QueryRow("SELECT role FROM users WHERE username = $1", username).Scan(&role)
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

	var data models.TeacherCoursesPageData

	// Получение количества курсов в БД
	var coursesCount int
	err = Db.QueryRow("SELECT COUNT(*) FROM courses").Scan(&coursesCount)
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
	courses := make([]models.Course, coursesCount)
	for i := 1; i <= coursesCount; i++ {
		var id int
		var name string
		err = Db.QueryRow("SELECT id, name FROM (SELECT *, ROW_NUMBER() OVER () as row_num FROM courses) AS subquery WHERE row_num = $1", i).Scan(&id, &name)
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
		courses[i-1] = models.Course{Id: id, Name: name}
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

	username := service.GetUsernameGromToken(token)
	log.Println("Получаем данные для пользователя: " + username)

	// Получение данных из БД
	var role string
	err = Db.QueryRow("SELECT role FROM users WHERE username = $1", username).Scan(&role)
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
	err = Db.QueryRow("SELECT COUNT(*) FROM groups").Scan(&groupsCount)
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
	err = Db.QueryRow("SELECT COUNT(*) FROM users").Scan(&usersCount)
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
	groups := make([]models.Group, groupsCount)
	for i := 1; i <= groupsCount; i++ {
		var id int
		var name string
		err = Db.QueryRow("SELECT id, name FROM (SELECT *, ROW_NUMBER() OVER () as row_num FROM groups) AS subquery WHERE row_num = $1", i).Scan(&id, &name)
		if err == sql.ErrNoRows {
			log.Println("Неправильные данные")
			sendError(w, "Неправильные данные", http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Println("Ошибка базы данных")
			sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
			return
		}
		groups[i-1] = models.Group{Id: id, Name: name}
	}

	// Считывание пользователей из БД
	users := make([]models.User, usersCount)
	for i := 1; i <= usersCount; i++ {
		var id int
		var name string
		var role string
		var group_id int
		err = Db.QueryRow("SELECT id, username, role, id_group FROM (SELECT *, ROW_NUMBER() OVER () as row_num FROM users) AS subquery WHERE row_num = $1", i).Scan(&id, &name, &role, &group_id)
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
		users[i-1] = models.User{Id: id, Username: name, Role: role, GroupName: groupName}
	}

	// log.Printf("Содержимое массива: ")
	// for i := 0; i < groupsCount; i++ {
	// 	h := groups[i]
	// 	log.Printf(strconv.Itoa(h.Id) + " " + h.Name + " ")
	// }

	adminData := models.AdminPanelData{
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

func getTestsData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	token, err := extractToken(w, r)
	if err != nil {
		log.Println("Токен не валиден. " + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Достаем информацию о тестах из БД
	username := service.GetUsernameGromToken(token)

	log.Println(username)

	var testsCount int
	err = Db.QueryRow(`SELECT DISTINCT COUNT(*)
						FROM users u
						JOIN groups g ON u.id_group = g.id
						JOIN groups_courses gc ON g.id = gc.id_group
						JOIN courses c ON gc.id_course = c.id
						JOIN tests t ON c.id = t.id_course
						WHERE u.username = $1`, username).Scan(&testsCount)
	if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	tests := make([]models.Test, testsCount)
	for i := 1; i <= testsCount; i++ {
		var id int
		var title string
		var upldate time.Time
		var enddate time.Time
		var duration int
		var attempts int
		err = Db.QueryRow(`WITH numbered_rows AS (
    							SELECT 
									t.id,
      							    t.name, 
        							t.upload_date, 
									t.ends_date,
        							t.duration,
									t.attempts,
        							ROW_NUMBER() OVER (ORDER BY t.upload_date DESC) as row_num
    							FROM users u
    							JOIN groups g ON u.id_group = g.id
    							JOIN groups_courses gc ON g.id = gc.id_group
    							JOIN courses c ON gc.id_course = c.id
    							JOIN tests t ON c.id = t.id_course
    							WHERE u.username = $1
							)
							SELECT id, name, upload_date, ends_date, duration, attempts
							FROM numbered_rows
							WHERE row_num = $2`, username, i).Scan(&id, &title, &upldate, &enddate, &duration, &attempts)
		if err == sql.ErrNoRows {
			log.Println("Неправильные данные")
			sendError(w, "Неправильные данные", http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Println("Ошибка базы данных")
			sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
			return
		}
		tests[i-1] = models.Test{ID: id, Title: title, UploadDate: upldate, EndDate: enddate, Duration: duration, Attempts: attempts}
	}

	testsData := models.TestsData{Tests: tests}

	// Устанавливаем заголовок Content-Type
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Кодируем данные в JSON и отправляем
	if err := json.NewEncoder(w).Encode(testsData); err != nil {
		http.Error(w, "Ошибка при формировании JSON", http.StatusInternalServerError)
		return
	}

	// jsonData, err := json.Marshal(testsData)
	// if err != nil {
	// 	log.Println("Ошибка в преобразовании в JSON")
	// 	sendError(w, "Внутренняя ошибка", http.StatusInternalServerError)
	// 	return
	// }

	// 	log.Println(utf8.RuneCountInString(string(jsonData)))
	// 	log.Println(string(jsonData))

	// 	// Отправляем JSON-ответ
	// 	w.WriteHeader(http.StatusOK)
	// 	json.NewEncoder(w).Encode(jsonData)
}

// Добавление пользователя через админ панель
func addUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var userData models.UserData

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
	err = Db.QueryRow("SELECT username FROM users WHERE username = $1", userData.Username).Scan(&checkuser)
	if err != nil {
		if err == sql.ErrNoRows {
			// Пользователя нет в базе, продолжаем
			log.Println("Проверка прошла успешно, пользователей с таким именем нет")
			// Ищем ID группы
			var id int
			err = Db.QueryRow("SELECT id FROM groups WHERE name = $1", userData.GroupName).Scan(&id)
			if err != nil {
				log.Println("Неправильные данные")
				sendError(w, "Неправильные данные", http.StatusUnauthorized)
			}

			hash, _ := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)

			_, err := Db.Exec("INSERT INTO users (username, password, role, id_group) VALUES ($1, $2, $3, $4)", userData.Username, string(hash), userData.Role, id)
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

	var groupData models.GroupData

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
	err = Db.QueryRow("SELECT name FROM groups WHERE name = $1", groupData.GroupName).Scan(&checkgroup)
	if err != nil {
		if err == sql.ErrNoRows {
			// Группы нет в базе, продолжаем
			_, err = Db.Exec("INSERT INTO groups (name) VALUES ($1)", groupData.GroupName)
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

	var user models.DeleteUserData

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
	err = Db.QueryRow("SELECT username FROM users WHERE username = $1", user.Name).Scan(&checkUser)
	if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	var delete string
	err = Db.QueryRow("DELETE FROM users WHERE username = $1", user.Name).Scan(&delete)
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

	var data models.DeleteGroupData

	// Чтение JSON
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Println("Некорректный JSON")
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	// Проверка группы в БД
	var checkGroup string
	err = Db.QueryRow("SELECT name FROM groups WHERE id = $1", data.Id).Scan(&checkGroup)
	if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Группа есть в базе

	_, err = Db.Exec("DELETE FROM groups WHERE id = $1", data.Id)
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

// извлекает тело HTTP-запроса как строку
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
	Db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Println("Database connection error:", err)
		return
	}
	//defer db.Close()
}

func main() {
	connectDB()

	log.Println("Сервер API запущен на " + port)

	r := mux.NewRouter()

	// Инициализация слоев приложения
	testRepo := repository.NewTestRepository(Db)
	testService := service.NewTestService(testRepo)
	testHandler := handler.NewTestHandler(testService)

	r.HandleFunc("/api/auth", handleAuth)
	r.HandleFunc("/api/verify", verifyToken)
	r.HandleFunc("/api/verifyadmin", verifyAdmin)
	r.HandleFunc("/api/verifyteacher", verifyTeacher)
	r.HandleFunc("/api/refreshtoken", refreshToken)
	r.HandleFunc("/api/getprofiledata", getProfileData)
	r.HandleFunc("/api/getteacherprofiledata", getTeacherProfileData)
	r.HandleFunc("/api/getteachercoursesdata", getTeacherCoursesData)
	r.HandleFunc("/api/getcoursesdata", getCoursesData)
	r.HandleFunc("/api/gettestsdata", getTestsData)

	r.HandleFunc("/api/admin/getadminpaneldata", getAdminPanelData)
	r.HandleFunc("/api/admin/adduser", addUser)
	r.HandleFunc("/api/admin/deleteuser", deleteUser)
	r.HandleFunc("/api/admin/addgroup", addGroup)
	r.HandleFunc("/api/admin/deletegroup", deleteGroup)

	// API tests-service
	r.HandleFunc("/api/tests/", testHandler.CreateTest)
	r.HandleFunc("/api/tests/{id}", testHandler.GetTest)
	r.HandleFunc("/api/tests/attempts", testHandler.StartAttempt)

	r.HandleFunc("/api/attempts/answers", testHandler.SubmitAnswer)
	r.HandleFunc("/api/attempts/finish", testHandler.FinishAttempt)

	// Запуск сервера (Ctrl + C, чтобы выключить)
	err := http.ListenAndServe(port, r)
	if err != nil {
		log.Fatal(err)
	}
}
