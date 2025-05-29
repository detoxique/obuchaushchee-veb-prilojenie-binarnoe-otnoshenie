package main

import (
	"api/internal/handler"
	"api/internal/models"
	"api/internal/repository"
	"api/internal/service"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gorilla/mux"

	"github.com/lib/pq"
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
	var role, group, id_group string
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

	err = Db.QueryRow("SELECT groups.name, groups.id FROM groups, users WHERE users.username = $1 AND users.id_group = groups.id", username).Scan(&group, &id_group)
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

	courses, err := GetUserCourses(Db, username, false)
	if err != nil {
		log.Println("Внутренняя ошибка")
		sendError(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	data := models.ProfilePageData{
		Username: username,
		Group:    group,
		Courses:  courses,
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

func GetUserCourses(db *sql.DB, username string, isTeacher bool) ([]models.Course, error) {
	if !isTeacher {
		groupID, err := GetUserGroupID(db, username)
		if err != nil {
			return nil, fmt.Errorf("error getting user group: %v", err)
		}

		courseIDs, err := GetUserCourseIDs(db, groupID)
		if err != nil {
			return nil, fmt.Errorf("error getting course IDs: %v", err)
		}

		if len(courseIDs) == 0 {
			return []models.Course{}, nil
		}

		courses, err := getCourses(db, courseIDs)
		if err != nil {
			return nil, err
		}

		filesMap, err := getFilesMap(db, courseIDs)
		if err != nil {
			return nil, err
		}

		testsMap, err := getTestsMap(db, courseIDs)
		if err != nil {
			return nil, err
		}

		for i := range courses {
			courseID := courses[i].Id
			courses[i].Files = getFromMap(filesMap, courseID)
			courses[i].Tests = getFromMap(testsMap, courseID)
		}
		return courses, nil
	} else {
		userID, err := GetUserID(db, username)
		if err != nil {
			return nil, fmt.Errorf("error getting user ID: %v", err)
		}

		courseIDs, err := GetTeacherCourseIDs(db, userID)
		if err != nil {
			return nil, fmt.Errorf("error getting course IDs: %v", err)
		}

		if len(courseIDs) == 0 {
			return []models.Course{}, nil
		}

		courses, err := getCourses(db, courseIDs)
		if err != nil {
			return nil, err
		}

		filesMap, err := getFilesMap(db, courseIDs)
		if err != nil {
			return nil, err
		}

		testsMap, err := getTestsMap(db, courseIDs)
		if err != nil {
			return nil, err
		}

		for i := range courses {
			courseID := courses[i].Id
			courses[i].Files = getFromMap(filesMap, courseID)
			courses[i].Tests = getFromMap(testsMap, courseID)
		}
		return courses, nil
	}
}

func GetUserGroupID(db *sql.DB, username string) (int, error) {
	var groupID sql.NullInt64
	err := db.QueryRow("SELECT id_group FROM users WHERE username = $1", username).Scan(&groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("user not found")
		}
		return 0, err
	}
	if !groupID.Valid {
		return 0, nil
	}
	return int(groupID.Int64), nil
}

func GetUserID(db *sql.DB, username string) (int, error) {
	var groupID sql.NullInt64
	err := db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("user not found")
		}
		return 0, err
	}
	if !groupID.Valid {
		return 0, nil
	}
	return int(groupID.Int64), nil
}

func GetUserCourseIDs(db *sql.DB, groupID int) ([]int, error) {
	if groupID == 0 {
		return []int{}, nil
	}

	rows, err := db.Query("SELECT id_course FROM groups_courses WHERE id_group = $1", groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courseIDs []int
	for rows.Next() {
		var courseID int
		if err := rows.Scan(&courseID); err != nil {
			return nil, err
		}
		courseIDs = append(courseIDs, courseID)
	}
	return courseIDs, rows.Err()
}

func GetTeacherCourseIDs(db *sql.DB, userID int) ([]int, error) {
	if userID == 0 {
		return []int{}, nil
	}

	rows, err := db.Query("SELECT id_course FROM users_courses WHERE id_user = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courseIDs []int
	for rows.Next() {
		var courseID int
		if err := rows.Scan(&courseID); err != nil {
			return nil, err
		}
		courseIDs = append(courseIDs, courseID)
	}
	return courseIDs, rows.Err()
}

func getCourses(db *sql.DB, courseIDs []int) ([]models.Course, error) {
	if len(courseIDs) == 0 {
		return []models.Course{}, nil
	}

	rows, err := db.Query("SELECT id, name FROM courses WHERE id = ANY($1)", pq.Array(courseIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var c models.Course
		if err := rows.Scan(&c.Id, &c.Name); err != nil {
			return nil, err
		}
		c.Files = []models.File{}
		c.Tests = []models.Test{}
		courses = append(courses, c)
	}
	return courses, rows.Err()
}

func getFilesMap(db *sql.DB, courseIDs []int) (map[int][]models.File, error) {
	filesMap := make(map[int][]models.File)
	if len(courseIDs) == 0 {
		return filesMap, nil
	}

	rows, err := db.Query(`
        SELECT id_course, name, filename, upload_date 
        FROM files 
        WHERE id_course = ANY($1)
    `, pq.Array(courseIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var f models.File
		var courseID int
		if err := rows.Scan(&courseID, &f.Name, &f.Filename, &f.UploadDate); err != nil {
			return nil, err
		}
		filesMap[courseID] = append(filesMap[courseID], f)
	}
	return filesMap, rows.Err()
}

func getTestsMap(db *sql.DB, courseIDs []int) (map[int][]models.Test, error) {
	testsMap := make(map[int][]models.Test)
	if len(courseIDs) == 0 {
		return testsMap, nil
	}

	rows, err := db.Query(`
        SELECT id, name, id_course, upload_date, ends_date, duration, attempts 
        FROM tests 
        WHERE id_course = ANY($1)
    `, pq.Array(courseIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t models.Test
		if err := rows.Scan(
			&t.ID,
			&t.Title,
			&t.CourseID,
			&t.UploadDate,
			&t.EndDate,
			&t.Duration,
			&t.Attempts,
		); err != nil {
			return nil, err
		}
		testsMap[t.CourseID] = append(testsMap[t.CourseID], t)
	}
	return testsMap, rows.Err()
}

func getFromMap[T any](m map[int][]T, key int) []T {
	if val, ok := m[key]; ok {
		return val
	}
	return []T{}
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

	courses, err := GetUserCourses(Db, username, true)
	if err != nil {
		log.Println("Внутренняя ошибка")
		sendError(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	data := models.ProfilePageData{
		Username: username,
		Group:    group,
		Courses:  courses,
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
		// var files []models.File
		// var tests []models.Test
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

		// Получаем файлы курса
		var filesCount int
		err = Db.QueryRow(`SELECT COUNT(*) FROM files WHERE id_course = $1`, id).Scan(&filesCount)
		if err != nil && err != sql.ErrNoRows {
			log.Println("Ошибка базы данных")
			sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
			return
		}

		files := make([]models.File, filesCount)
		for j := 1; j <= filesCount; j++ {
			var file_id int
			var name string
			var upload time.Time
			err = Db.QueryRow(`WITH numbered_rows AS (
									SELECT id, name, upload_date, ROW_NUMBER() OVER (ORDER BY id DESC) as row_num
								FROM files
								WHERE id_course = $1
							)
							SELECT id, name, upload_date
							FROM numbered_rows
							WHERE row_num = $2`, id, j).Scan(&file_id, &name, &upload)
			if err != nil && err != sql.ErrNoRows {
				log.Println("Ошибка базы данных")
				sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
				return
			}
			files[j-1] = models.File{Id: file_id, Name: name, UploadDate: upload}
		}

		// Получаем тесты курса
		var testsCount int
		err = Db.QueryRow(`SELECT COUNT(*) FROM tests WHERE id_course = $1`, id).Scan(&testsCount)
		if err != nil && err != sql.ErrNoRows {
			log.Println("Ошибка базы данных")
			sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
			return
		}

		tests := make([]models.Test, testsCount)
		for j := 1; j <= testsCount; j++ {
			var test_id int
			var name string
			var end_date time.Time
			var duration int
			err = Db.QueryRow(`WITH numbered_rows AS (
									SELECT id, name, ends_date, duration, ROW_NUMBER() OVER (ORDER BY id DESC) as row_num
								FROM tests
								WHERE id_course = $1
							)
							SELECT id, name, ends_date, duration
							FROM numbered_rows
							WHERE row_num = $2`, id, j).Scan(&test_id, &name, &end_date, &duration)
			if err != nil && err != sql.ErrNoRows {
				log.Println("Ошибка базы данных")
				sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
				return
			}
			tests[j-1] = models.Test{ID: test_id, Title: name, EndDate: end_date, Duration: duration}
		}

		courses[i-1] = models.Course{Id: id, Name: name, Files: files, Tests: tests}
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

	var data models.CoursesPageData

	courses, err := GetUserCourses(Db, username, false)
	if err != nil {
		log.Println("Ошибка получения данных курсов из БД")
		sendError(w, "Внутренняя ошибка", http.StatusInternalServerError)
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

func getCourseData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	name := vars["name"]

	var courseID string
	err := Db.QueryRow("SELECT id FROM courses WHERE name = $1", name).Scan(&courseID)
	if err == sql.ErrNoRows {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	var filesCount int
	err = Db.QueryRow("SELECT COUNT(*) FROM files WHERE id_course = $1", courseID).Scan(&filesCount)
	if err == sql.ErrNoRows {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	files := make([]models.File, filesCount)
	for i := 1; i <= filesCount; i++ {
		var name string
		var upload_time time.Time
		var filename string
		err = Db.QueryRow("SELECT name, upload_date, filename FROM (SELECT *, ROW_NUMBER() OVER () as row_num FROM files WHERE id_course = $1) AS subquery WHERE row_num = $2", courseID, i).Scan(&name, &upload_time, &filename)
		if err == sql.ErrNoRows {
			log.Println("Неправильные данные")
			sendError(w, "Неправильные данные", http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Println("Ошибка базы данных")
			sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
			return
		}
		files[i-1] = models.File{Name: name, Filename: filename, UploadDate: upload_time}
	}

	var testsCount int
	err = Db.QueryRow("SELECT COUNT(*) FROM tests WHERE id_course = $1", courseID).Scan(&testsCount)
	if err == sql.ErrNoRows {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	tests := make([]models.Test, testsCount)
	for i := 1; i <= testsCount; i++ {
		var title string
		var EndDate time.Time
		err = Db.QueryRow("SELECT name, ends_date FROM (SELECT *, ROW_NUMBER() OVER () as row_num FROM tests WHERE id_course = $1) AS subquery WHERE row_num = $2", courseID, i).Scan(&title, &EndDate)
		if err == sql.ErrNoRows {
			log.Println("Неправильные данные")
			sendError(w, "Неправильные данные", http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Println("Ошибка базы данных")
			sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
			return
		}
		tests[i-1] = models.Test{Title: title, EndDate: EndDate}
	}
	course := models.Course{
		Files: files,
		Tests: tests,
	}

	data := models.ServeCoursePage{
		Course: course,
	}
	log.Println("Количество тестов: " + strconv.Itoa(len(tests)))

	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func getViewData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	name := vars["name"]

	var filename string
	err := Db.QueryRow("SELECT filename FROM files WHERE name = $1", name).Scan(&filename)
	if err == sql.ErrNoRows {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(filename)
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

func handleCreateCourse(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен запрос на создание курса")
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Name        string `json:"name"`
		Description string `json:"desription"`
		Groups      []int  `json:"groups"`
		Token       string `json:"access_token"`
	}

	// Чтение JSON
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Println("Некорректный JSON " + err.Error())
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	tokenCheck, err := service.CheckAccessToken(data.Token)
	if err != nil {
		log.Println("Токен не валиден. " + err.Error())
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	if !tokenCheck {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
	}

	username := service.GetUsernameGromToken(data.Token)
	log.Println("Получаем данные для пользователя: " + username)

	// Получение данных из БД
	var role, user_id string
	err = Db.QueryRow("SELECT role, id FROM users WHERE username = $1", username).Scan(&role, &user_id)
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
	if role != "teacher" {
		log.Println("Пользователь не является преподавателем")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	}

	// Прошел проверку
	// Проверка курса в БД
	var checkCourse string
	var course_id int
	err = Db.QueryRow("SELECT name FROM courses WHERE name = $1", data.Name).Scan(&checkCourse)
	if err != nil {
		if err == sql.ErrNoRows {
			// курса нет в базе, продолжаем

			err = Db.QueryRow("INSERT INTO courses (name) VALUES ($1) RETURNING id", data.Name).Scan(&course_id)
			if err != nil {
				log.Println("Ошибка базы данных")
				sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
				return
			}

			_, err = Db.Exec("INSERT INTO users_courses (id_user, id_course) VALUES ($1, $2)", user_id, course_id)
			if err != nil {
				log.Println("Ошибка базы данных")
				sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
				return
			}

			for i := 0; i < len(data.Groups); i++ {
				_, err = Db.Exec("INSERT INTO groups_courses (id_group, id_course) VALUES ($1, $2)", data.Groups[i], course_id)
				if err != nil {
					log.Println("Ошибка базы данных " + err.Error())
					sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
					return
				}
			}
		} else {
			log.Println("Ошибка базы данных")
			sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	returnCourse := models.Course{Id: course_id, Name: data.Name}
	json.NewEncoder(w).Encode(returnCourse)
}

func handleDeleteCourse(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен запрос на удаление курса")
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Id    string `json:"id"`
		Token string `json:"token"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Println("Некорректный JSON")
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	tokenCheck, err := service.CheckAccessToken(data.Token)
	if err != nil {
		log.Println("Токен не валиден. " + err.Error())
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	if !tokenCheck {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
	}

	username := service.GetUsernameGromToken(data.Token)
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

	if role != "teacher" {
		log.Println("Неправильные данные")
		sendError(w, "Неправильные данные", http.StatusUnauthorized)
		return
	}

	// Проверка пройдена

	_, err = Db.Exec("DELETE FROM users_courses WHERE id_course = $1", data.Id)
	if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = Db.Exec("DELETE FROM groups_courses WHERE id_course = $1", data.Id)
	if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = Db.Exec("DELETE FROM courses WHERE id = $1", data.Id)
	if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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

func getCourseNameByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Println("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var id string

	// Чтение JSON
	err := json.NewDecoder(r.Body).Decode(&id)
	if err != nil {
		log.Println("Некорректный JSON")
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	// Поиск курса в БД
	var coursename string
	err = Db.QueryRow("SELECT name FROM courses WHERE id = $1", id).Scan(&coursename)
	if err != nil {
		log.Println("Ошибка базы данных")
		sendError(w, "Ошибка базы данных "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(coursename)
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
	r.HandleFunc("/api/getcoursenamebyid", getCourseNameByID)
	r.HandleFunc("/api/gettestsdata", getTestsData)
	r.HandleFunc("/api/getcoursedata/{name}", getCourseData)
	r.HandleFunc("/api/getviewdata/{name}", getViewData)
	r.HandleFunc("/api/createcourse", handleCreateCourse)
	r.HandleFunc("/api/deletecourse", handleDeleteCourse)

	r.HandleFunc("/api/admin/getadminpaneldata", getAdminPanelData)
	r.HandleFunc("/api/admin/adduser", addUser)
	r.HandleFunc("/api/admin/deleteuser", deleteUser)
	r.HandleFunc("/api/admin/addgroup", addGroup)
	r.HandleFunc("/api/admin/deletegroup", deleteGroup)

	// API tests-service
	r.HandleFunc("/api/tests/", testHandler.CreateTest)
	r.HandleFunc("/api/tests/test/{id}", testHandler.GetTest)
	r.HandleFunc("/api/tests/attempts", testHandler.StartAttempt)

	r.HandleFunc("/api/attempts/answers", testHandler.SubmitAnswer)
	r.HandleFunc("/api/attempts/finish", testHandler.FinishAttempt)

	// Запуск сервера (Ctrl + C, чтобы выключить)
	err := http.ListenAndServe(port, r)
	if err != nil {
		log.Fatal(err)
	}
}
