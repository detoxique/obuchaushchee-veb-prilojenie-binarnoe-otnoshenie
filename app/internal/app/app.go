package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"
	"time"
)

// Данные для входа
type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserData struct {
	Username  string `json:"Username"`
	Password  string `json:"Password"`
	Role      string `json:"Role"`
	GroupName string `json:"GroupName"`
}

type GroupData struct {
	GroupName string `json:"GroupName"`
}

type TokenResponse struct {
	AccessToken string `json:"Authorization"`
}

type ProfilePageData struct {
	Username string `json:"Username"`
	Group    string `json:"Group"`
}

type TeacherCoursesPageData struct {
	Courses []string `json:"Courses"`
}

type Course struct {
	Name  string `json:"Name"`
	Files []File `json:"Files"`
	Tests []Test `json:"Tests"`
}

type File struct {
	Name       string    `json:"Name"`
	UploadDate time.Time `json:"UploadDate"`
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

// Данные на админ панели(группы и пользователи)
type AdminPanelData struct {
	Groups []Group `json:"Groups"`
	Users  []User  `json:"Users"`
}

type ServeAdminPanelData struct {
	Groups                  template.HTML `json:"Groups"`
	GroupsTable             template.HTML `json:"GroupsTable"`
	UsersTable              template.HTML `json:"UsersTable"`
	ВсегоПосещений          int64         `json:"ВсегоПосещений"`
	СамаяПопулярнаяСтраница string        `json:"СамаяПопулярнаяСтраница"`
}

type Statistics struct {
	ПосещенияПрофль      int64 `json:"ПосещенияПрофль"`
	ПосещенияАдминПанель int64 `json:"ПосещенияАдминПанель"`
	ПосещенияОценки      int64 `json:"ПосещенияОценки"`
	ПосещенияКурсы       int64 `json:"ПосещенияКурсы"`
}

type StatsToView struct {
	ВсегоПосещений          int64  `json:"ВсегоПосещений"`
	СамаяПопулярнаяСтраница string `json:"СамаяПопулярнаяСтраница"`
}

var Stats Statistics

type CoursesPageServeData struct {
	Courses template.HTML `json:"courses"`
}

type DeleteGroupData struct {
	Id string `json:"Id"`
}

type DeleteUserData struct {
	Token string `json:"token"`
	Name  string `json:"Username"`
}

type DeleteUser struct {
	Name string `json:"Username"`
}

// Тесты
type Test struct {
	Title      string    `json:"Title"`
	UploadDate time.Time `json:"UploadDate"`
	EndDate    time.Time `json:"EndDate"`
	Duration   string    `json:"Duration"`
}

type TestsData struct {
	Tests []Test `json:"Tests"`
}

// Страница авторизации
func serveLoginPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// Страница Профиля
func serveProfilePage(w http.ResponseWriter, r *http.Request) {
	Stats.ПосещенияПрофль++

	tmpl, err := template.ParseFiles("templates/profile.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)

	UpdateStats()
}

// Страница Профиля
func serveCoursesPage(w http.ResponseWriter, r *http.Request) {
	Stats.ПосещенияКурсы++

	tmpl, err := template.ParseFiles("templates/courses.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)

	UpdateStats()
}

// Страница оценок
func serveMarksPage(w http.ResponseWriter, r *http.Request) {
	Stats.ПосещенияОценки++

	tmpl, err := template.ParseFiles("templates/marks.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)

	UpdateStats()
}

// Страница авторизации
func serveNotificationsPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/notifications.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// Страница админ панели
func serveAdminPage(w http.ResponseWriter, r *http.Request) {
	// TODO: Проверять, авторизован ли пользователь в учетку админа

	Stats.ПосещенияАдминПанель++

	var mostPopular string
	if Stats.ПосещенияАдминПанель > Stats.ПосещенияПрофль && Stats.ПосещенияАдминПанель > Stats.ПосещенияОценки && Stats.ПосещенияАдминПанель > Stats.ПосещенияКурсы {
		mostPopular = "Админ Панель"
	} else if Stats.ПосещенияПрофль > Stats.ПосещенияАдминПанель && Stats.ПосещенияПрофль > Stats.ПосещенияОценки && Stats.ПосещенияПрофль > Stats.ПосещенияКурсы {
		mostPopular = "Профиль"
	} else if Stats.ПосещенияОценки > Stats.ПосещенияПрофль && Stats.ПосещенияОценки > Stats.ПосещенияАдминПанель && Stats.ПосещенияОценки > Stats.ПосещенияКурсы {
		mostPopular = "Оценки"
	} else {
		mostPopular = "Курсы"
	}

	stat := StatsToView{
		ВсегоПосещений:          Stats.ПосещенияАдминПанель + Stats.ПосещенияОценки + Stats.ПосещенияПрофль + Stats.ПосещенияКурсы,
		СамаяПопулярнаяСтраница: mostPopular,
	}

	tmpl, err := template.ParseFiles("templates/admin.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, stat)

	UpdateStats()
}

// Получение страницы профиля с данными
func getAdminPanelData(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на получение данных админ панели")
	// Принимаются только POST запросы
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Проверка токена
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	token := ExtractJWT(string(dump))
	if token == "" {
		slog.Info("Не удалось вытащить токен из запроса")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	slog.Info("Отправлен запрос на подтверждение токена")

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/verifyadmin", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера: " + (string)(resp.StatusCode))
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Проверка прошла успешно
	slog.Info("Проверка админа прошла успешно")

	// Отправка запроса для получения данных
	resp, err = http.Post("http://localhost:1337/api/admin/getadminpaneldata", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера: " + (string)(resp.StatusCode))
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Успешный ответ
	var adminData AdminPanelData

	err = json.NewDecoder(resp.Body).Decode(&adminData)
	if err != nil {
		slog.Info("Не удалось считать данные профиля")
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	var mostPopular string
	if Stats.ПосещенияАдминПанель > Stats.ПосещенияПрофль && Stats.ПосещенияАдминПанель > Stats.ПосещенияОценки && Stats.ПосещенияАдминПанель > Stats.ПосещенияКурсы {
		mostPopular = "Админ Панель"
	} else if Stats.ПосещенияПрофль > Stats.ПосещенияАдминПанель && Stats.ПосещенияПрофль > Stats.ПосещенияОценки && Stats.ПосещенияПрофль > Stats.ПосещенияКурсы {
		mostPopular = "Профиль"
	} else if Stats.ПосещенияОценки > Stats.ПосещенияПрофль && Stats.ПосещенияОценки > Stats.ПосещенияАдминПанель && Stats.ПосещенияОценки > Stats.ПосещенияКурсы {
		mostPopular = "Оценки"
	} else {
		mostPopular = "Курсы"
	}

	// selector HTML
	var sel string
	var groupsTable string
	for i := 0; i < len(adminData.Groups); i++ {
		sel += `<option value="` + adminData.Groups[i].Name + `">` + adminData.Groups[i].Name + `</option>`
		if adminData.Groups[i].Name != "admins" && adminData.Groups[i].Name != "teachers" {
			groupsTable += `<tr><td>` + strconv.Itoa(adminData.Groups[i].Id) + `</td>`
			groupsTable += `<td>` + adminData.Groups[i].Name + `</td>`
			groupsTable += `<td><button type="button" id="delete-idgroup-` + strconv.Itoa(adminData.Groups[i].Id) + `" class="btn btn-outline-danger btn-sm">Удалить</button></td></tr>`
		}
	}

	// table users HTML
	var usersTable string
	for i := 0; i < len(adminData.Users); i++ {
		usersTable += `<tr><td>` + adminData.Users[i].Username + `</td>`
		// role selector
		usersTable += `<td><select><option value="`
		if adminData.Users[i].Role == "student" {
			usersTable += `student">Студент</option><option value="teacher">Преподаватель</option>
                                <option value="admin">Админ</option>`
		} else if adminData.Users[i].Role == "teacher" {
			usersTable += `teacher">Преподаватель</option><option value="student">Студент</option>
                                <option value="admin">Админ</option>`
		} else {
			usersTable += `admin">Админ</option><option value="student">Студент</option>
                                <option value="teacher">Преподаватель</option>`
		}
		usersTable += `</select></td>`
		// group selector
		usersTable += `<td>
                            <select>
                                <option value="` + adminData.Users[i].GroupName + `">` + adminData.Users[i].GroupName + `</option>`
		for j := 0; j < len(adminData.Groups); j++ {
			if adminData.Groups[j].Name == adminData.Users[i].GroupName {
				continue
			}
			usersTable += `<option value="` + adminData.Groups[j].Name + `">` + adminData.Groups[j].Name + `</option>`
		}
		usersTable += `</select>
                        </td>`
		// button
		usersTable += `<td><button type="button" id="delete-user-` + adminData.Users[i].Username + `" class="btn btn-outline-danger btn-sm">Удалить</button></td></tr>`
	}

	data := ServeAdminPanelData{
		Groups:         template.HTML(sel),
		GroupsTable:    template.HTML(groupsTable),
		UsersTable:     template.HTML(usersTable),
		ВсегоПосещений: Stats.ПосещенияАдминПанель + Stats.ПосещенияОценки + Stats.ПосещенияПрофль + Stats.ПосещенияКурсы,
		СамаяПопулярнаяСтраница: mostPopular,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles("templates/admin.html")
	if err != nil {
		slog.Info("Не удалось получить шаблон страницы профиля" + err.Error())
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		slog.Info(err.Error())
	}
}

// Получение страницы профиля с данными
func getProfileData(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на получение данных профиля")
	// Принимаются только GET запросы
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Проверка токена
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	token := ExtractJWT(string(dump))
	if token == "" {
		slog.Info("Не удалось вытащить токен из запроса")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	slog.Info("Отправлен запрос на подтверждение токена")

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/verify", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера: " + (string)(resp.StatusCode))
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Получение данных

	slog.Info("Токен валиден. Отправлен запрос на получение данных")
	// Отправка запроса на другой сервер
	respData, err := http.Post("http://localhost:1337/api/getprofiledata", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer respData.Body.Close()

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if respData.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера авторизации")
		body, _ := io.ReadAll(respData.Body)
		w.WriteHeader(respData.StatusCode)
		w.Write(body)
		return
	}

	slog.Info("Получен успешный ответ")

	// Успешный ответ
	var profileData ProfilePageData

	err = json.NewDecoder(respData.Body).Decode(&profileData)
	if err != nil {
		slog.Info("Не удалось считать данные профиля")
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	// Отправление страницы пользователю
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles("templates/profile.html")
	if err != nil {
		slog.Info("Не удалось получить шаблон страницы профиля")
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, profileData)
}

// Получение страницы профиля с данными
func getTeacherProfileData(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на получение данных профиля")
	// Принимаются только GET запросы
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Проверка токена
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	token := ExtractJWT(string(dump))
	if token == "" {
		slog.Info("Не удалось вытащить токен из запроса")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	slog.Info("Отправлен запрос на подтверждение токена")

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/verify", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера: " + (string)(resp.StatusCode))
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Получение данных

	slog.Info("Токен валиден. Отправлен запрос на получение данных")
	// Отправка запроса на другой сервер
	respData, err := http.Post("http://localhost:1337/api/getteacherprofiledata", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer respData.Body.Close()

	slog.Info("Получен успешный ответ")

	// Успешный ответ
	var profileData ProfilePageData

	err = json.NewDecoder(respData.Body).Decode(&profileData)
	if err != nil {
		slog.Info("Не удалось считать данные профиля")
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	// Отправление страницы пользователю
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles("templates/profileteacher.html")
	if err != nil {
		slog.Info("Не удалось получить шаблон страницы профиля " + err.Error())
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, profileData)
}

func getTeacherCoursesData(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на получение данных курсов преподавателя")
	// Принимаются только GET запросы
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Проверка токена
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	token := ExtractJWT(string(dump))
	if token == "" {
		slog.Info("Не удалось вытащить токен из запроса")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	slog.Info("Отправлен запрос на подтверждение токена")

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/verifyteacher", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера: " + (string)(resp.StatusCode))
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Получение данных

	slog.Info("Токен валиден. Отправлен запрос на получение данных")
	// Отправка запроса на другой сервер
	respData, err := http.Post("http://localhost:1337/api/getteachercoursesdata", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer respData.Body.Close()

	slog.Info("Получен успешный ответ")

	// Успешный ответ
	var coursesData TeacherCoursesPageData

	err = json.NewDecoder(respData.Body).Decode(&coursesData)
	if err != nil {
		slog.Info("Не удалось считать данные профиля")
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	// Отправление страницы пользователю
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles("templates/coursesteacher.html")
	if err != nil {
		slog.Info("Не удалось получить шаблон страницы профиля " + err.Error())
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, coursesData)
}

func getCoursesData(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на получение данных курсов преподавателя")
	// Принимаются только POST запросы
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Проверка токена
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	token := ExtractJWT(string(dump))
	if token == "" {
		slog.Info("Не удалось вытащить токен из запроса")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	slog.Info("Отправлен запрос на подтверждение токена")

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/verify", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера: " + strconv.Itoa(resp.StatusCode))
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Получение данных

	slog.Info("Токен валиден. Отправлен запрос на получение данных")
	// Отправка запроса на другой сервер
	respData, err := http.Post("http://localhost:1337/api/getcoursesdata", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer respData.Body.Close()

	slog.Info("Получен успешный ответ")

	// Успешный ответ
	var coursesData CoursesPageData

	err = json.NewDecoder(respData.Body).Decode(&coursesData)
	if err != nil {
		slog.Info("Не удалось считать данные профиля" + err.Error())
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	slog.Info("Количество курсов: " + strconv.Itoa(len(coursesData.Courses)))

	coursesHTML := `<ul class="courses-list">`

	for i := 0; i < len(coursesData.Courses); i++ {
		coursesHTML += `<li><a href="#">` + coursesData.Courses[i] + `</a><br><h4>1 Тест до 01.02.2025!</h4></li>`
	}

	coursesHTML += `</ul>`

	var data CoursesPageServeData

	data.Courses = template.HTML(coursesHTML)

	slog.Info(coursesHTML)

	// Отправление страницы пользователю
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles("templates/courses.html")
	if err != nil {
		slog.Info("Не удалось получить шаблон страницы профиля " + err.Error())
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		slog.Info(err.Error())
	}
}

func getTeacherMarksData(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на получение данных успеваемости для преподавателя")
	// Принимаются только POST запросы
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Проверка токена
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	token := ExtractJWT(string(dump))
	if token == "" {
		slog.Info("Не удалось вытащить токен из запроса")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	slog.Info("Отправлен запрос на подтверждение токена")

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/verifyteacher", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера: " + strconv.Itoa(resp.StatusCode))
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Проверка прошла успешно
	// Отправление страницы пользователю
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles("templates/marksteacher.html")
	if err != nil {
		slog.Info("Не удалось получить шаблон страницы успеваемости " + err.Error())
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		slog.Info(err.Error())
	}
}

func getMarksData(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на получение данных успеваемости для студента")
	// Принимаются только POST запросы
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Проверка токена
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	token := ExtractJWT(string(dump))
	if token == "" {
		slog.Info("Не удалось вытащить токен из запроса")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	slog.Info("Отправлен запрос на подтверждение токена")

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/verify", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера: " + strconv.Itoa(resp.StatusCode))
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Проверка прошла успешно
	// Отправление страницы пользователю
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles("templates/marks.html")
	if err != nil {
		slog.Info("Не удалось получить шаблон страницы успеваемости " + err.Error())
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		slog.Info(err.Error())
	}
}

func getTestsData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	token := ExtractJWT(string(dump))
	if token == "" {
		slog.Info("Не удалось вытащить токен")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	slog.Info("Токен: " + token)

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/gettestsdata", "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Info("Ошибка получения данных тестов http")
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера")
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}
	slog.Info("Статус 200")

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		slog.Info("Ошибка при чтении тела ответа")
	}
	slog.Info(string(body))

	// var tests TestsData

	// err = json.NewDecoder(resp.Body).Decode(&tests)
	// if err != nil {
	// 	slog.Info("Ошибка при чтении JSON")
	// 	w.Header().Set("Content-Type", "application/json")
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	w.Write(body)
	// 	return
	// }
	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

// Авторизация
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var loginData LoginData

	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		slog.Info("Не удалось считать данные для входа")
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(loginData)
	if err != nil {
		slog.Info("Не удалось создать JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/auth", "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Info("Не удалось отправить запрос. Ошибка сервера авторизации")
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера авторизации")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Успешный ответ
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

func handleVerifyToken(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на подтверждение токена")
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	token := ExtractJWT(string(dump))
	if token == "" {
		slog.Info("Не удалось вытащить токен")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/verify", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}
	w.Write(body)
}

func handleVerifyAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Проверка токена
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	token := ExtractJWT(string(dump))
	if token == "" {
		slog.Info("Не удалось вытащить токен из запроса")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	slog.Info("Отправлен запрос на подтверждение токена")

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/verifyadmin", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера: " + (string)(resp.StatusCode))
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Проверка прошла успешно
	slog.Info("Проверка админа прошла успешно")

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера авторизации")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Успешный ответ
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

func handleVerifyTeacher(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на подтверждение входа препода")
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Проверка токена
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	token := ExtractJWT(string(dump))
	if token == "" {
		slog.Info("Не удалось вытащить токен из запроса")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	slog.Info("Отправлен запрос на подтверждение токена")

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/verifyteacher", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера: " + (string)(resp.StatusCode))
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Проверка прошла успешно
	slog.Info("Проверка препода прошла успешно")

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера авторизации")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Успешный ответ
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

func handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Вытаскиваем токен из запроса
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	token := ExtractJWT(string(dump))
	if token == "" {
		slog.Info("Ошибка озвлечения токена")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/refreshtoken", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}
	w.Write(body)
}

func handleUploadFile(w http.ResponseWriter, r *http.Request) {

}

// Добавление пользователей
func handleAddUser(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на добавление пользователя в БД")
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	slog.Info(string(dump))

	var userData UserData

	err = json.NewDecoder(r.Body).Decode(&userData)
	if err != nil {
		slog.Info("Не удалось считать данные для входа")
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(userData)
	if err != nil {
		slog.Info("Не удалось создать JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/admin/adduser", "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Info("Не удалось отправить запрос. Ошибка сервера авторизации")
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Успешный ответ
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

// Удаление пользователей
func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		return
	}

	slog.Info(string(dump))

	var userData DeleteUserData

	err = json.NewDecoder(r.Body).Decode(&userData)
	if err != nil {
		slog.Info("Не удалось считать данные для входа")
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	slog.Info("Удаляем пользователя " + userData.Name)

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&userData.Token)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/verifyadmin", "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Info("Не удалось отправить запрос. Ошибка сервера авторизации")
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Проверка админа прошла успешно
	// Подготовка запроса к другому серверу
	var delUser DeleteUser
	delUser.Name = userData.Name
	body, err = json.Marshal(&delUser)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Отправка запроса на другой сервер
	resp, err = http.Post("http://localhost:1337/api/admin/deleteuser", "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Info("Не удалось отправить запрос. Ошибка сервера авторизации")
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Успешный ответ
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

// Добавление групп
func handleAddGroup(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на добавление группы в БД")
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var groupData GroupData

	err := json.NewDecoder(r.Body).Decode(&groupData)
	if err != nil {
		slog.Info("Не удалось считать данные для входа")
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(groupData)
	if err != nil {
		slog.Info("Не удалось создать JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/admin/addgroup", "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Info("Не удалось отправить запрос. Ошибка сервера авторизации")
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Успешный ответ
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

// Удаление групп
func handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на удаление группы из БД")
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var data DeleteGroupData

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		slog.Info("Не удалось считать данные для входа")
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	slog.Info(data.Id)

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(data)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/admin/deletegroup", "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Info("Не удалось отправить запрос. Ошибка сервера авторизации")
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Успешный ответ
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

// Изменение группы пользователя
func handleChangeUserGroup(w http.ResponseWriter, r *http.Request) {

}

// Изменение роли пользователя
func handleChangeUserRole(w http.ResponseWriter, r *http.Request) {

}

// Извлекает JWT-токен из строки HTTP-запроса
func ExtractJWT(request string) string {
	// Разбиваем запрос на строки
	lines := strings.Split(request, "\n")

	// Ищем заголовок Authorization
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "authorization:") {
			// Удаляем префикс "Authorization:" и пробелы
			auth := strings.TrimSpace(strings.TrimPrefix(line, "Authorization:"))
			return auth
		}
	}

	return ""
}

// Обновить файл статистики
func UpdateStats() {
	// Сериализация в JSON (с форматированием)
	jsonData, err := json.MarshalIndent(Stats, "", "  ")
	if err != nil {
		panic(err)
	}

	// Запись в файл "stats.json"
	err = os.WriteFile("stats.json", jsonData, 0644) // Права 0644 (rw-r--r--)
	if err != nil {
		panic(err)
	}
}

func Run(ctx context.Context) error {
	slog.Info("Сервер запущен. Порт: 8080")

	s := http.Server{
		Addr: ":8080",
	}

	// Загрузка статистики

	// Чтение файла
	fileData, err := os.ReadFile("stats.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(fileData, &Stats)
	if err != nil {
		panic(err)
	}

	// HTML
	http.HandleFunc("/", serveLoginPage)
	http.HandleFunc("/profile", serveProfilePage)
	http.HandleFunc("/marks", serveMarksPage)
	http.HandleFunc("/admin", serveAdminPage)
	http.HandleFunc("/courses", serveCoursesPage)
	http.HandleFunc("/notifications", serveNotificationsPage)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// API
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/verify", handleVerifyToken)
	http.HandleFunc("/api/verifyadmin", handleVerifyAdmin)
	http.HandleFunc("/api/verifyteacher", handleVerifyTeacher)
	http.HandleFunc("/api/refreshtoken", handleRefreshToken)
	http.HandleFunc("/api/getprofiledata", getProfileData)
	http.HandleFunc("/api/getteacherprofiledata", getTeacherProfileData)
	http.HandleFunc("/api/getadminpaneldata", getAdminPanelData)
	http.HandleFunc("/api/getteachercoursesdata", getTeacherCoursesData)
	http.HandleFunc("/api/getcoursesdata", getCoursesData)
	http.HandleFunc("/api/getteachermarksdata", getTeacherMarksData)
	http.HandleFunc("/api/getmarksdata", getMarksData)
	http.HandleFunc("/api/gettestsdata", getTestsData)
	http.HandleFunc("/api/upload", handleUploadFile)

	http.HandleFunc("/api/admin/adduser", handleAddUser)
	http.HandleFunc("/api/admin/deleteuser", handleDeleteUser)
	http.HandleFunc("/api/admin/addgroup", handleAddGroup)
	http.HandleFunc("/api/admin/deletegroup", handleDeleteGroup)

	http.HandleFunc("/api/admin/changeusergroup", handleChangeUserGroup)
	http.HandleFunc("/api/admin/changeuserrole", handleChangeUserRole)

	go func() {
		<-ctx.Done()
		s.Shutdown(ctx)
	}()

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
