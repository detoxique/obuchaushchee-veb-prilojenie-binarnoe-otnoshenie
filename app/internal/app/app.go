package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"text/template"
)

// Данные для входа
type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken string `json:"Authorization"`
}

type ProfilePageData struct {
	Username              string `json:"Username"`
	Role                  string `json:"Role"`
	Group                 string `json:"Group"`
	PerformancePercentage int    `json:"PerformancePersentage"`
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
	GroupsForSelector  string
	GroupsHTML         string
	GroupsHTMLSelector string
	UsersHTML          string
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
	var adminHTML ServeAdminPanelData

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

	stat := StatsToView{
		ВсегоПосещений:          Stats.ПосещенияАдминПанель + Stats.ПосещенияОценки + Stats.ПосещенияПрофль + Stats.ПосещенияКурсы,
		СамаяПопулярнаяСтраница: mostPopular,
	}

	// Создание HTML кода для вставки
	page := `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Admin Panel</title>
    <style>
        body {
            font-family: sans-serif;
            margin: 0;
            padding: 0;
            background-color: #f4f4f4;
            color: #333;
        }

        .container {
            width: 80%;
            margin: 20px auto;
            background-color: #fff;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
        }

        h1 {
            text-align: center;
            margin-bottom: 20px;
        }

        nav {
            background-color: #333;
            color: #fff;
            padding: 10px;
            margin-bottom: 20px;
        }

        nav ul {
            list-style: none;
            padding: 0;
            margin: 0;
            display: flex;
            justify-content: space-around;
        }

        nav a {
            color: #fff;
            text-decoration: none;
            padding: 10px 15px;
            border-radius: 5px;
            transition: background-color 0.3s;
        }

        nav a:hover {
            background-color: #555;
        }

        .form-section, .groups-add-section, .groups-section, .users-section, .backup-section, .stats-section {
            margin-bottom: 20px;
            border: 1px solid #ddd;
            padding: 15px;
            border-radius: 5px;
            background-color: #f9f9f9;
        }

        .form-section h2, .groups-add-section h2, .groups-section h2, .users-section h2, .backup-section h2, .stats-section h2 {
            margin-top: 0;
            margin-bottom: 15px;
        }

        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }

        input[type="login"], input[type="password"], input[type="groupname"], select[type="role"], select[type="group"] {
            width: 100%;
            padding: 8px;
            margin-bottom: 10px;
            border: 1px solid #ccc;
            border-radius: 4px;
            box-sizing: border-box;
        }

        button {
            background-color: #4CAF50;
            color: white;
            padding: 10px 15px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            transition: background-color 0.3s;
        }

        button:hover {
            background-color: #3e8e41;
        }

        .groups-list {
            list-style: none;
            padding: 0;
            margin: 0;
        }

        .groups-list li {
            padding: 10px;
            border-bottom: 1px solid #eee;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .groups-list li:last-child {
            border-bottom: none;
        }

        .user-list {
            list-style: none;
            padding: 0;
            margin: 0;
        }

        .user-list li {
            padding: 10px;
            border-bottom: 1px solid #eee;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .user-list li:last-child {
            border-bottom: none;
        }

        .delete-button {
            background-color: #f44336;
            color: white;
            padding: 8px 12px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            transition: background-color 0.3s;
        }

        .delete-button:hover {
            background-color: #d32f2f;
        }

        .stats-section ul {
            list-style: none;
            padding: 0;
            margin: 0;
        }

        .stats-section li {
            padding: 5px 0;
        }

        /* Responsive Design */
        @media (max-width: 600px) {
            .container {
                width: 95%;
            }

            nav ul {
                flex-direction: column;
                align-items: center;
            }

            nav li {
                margin-bottom: 5px;
            }
        }
    </style>
</head>
<body>

    <div class="container">
        <h1>Админ панель</h1>

        <nav>
            <ul>
                <li><a href="#add-user">Добавить пользователя</a></li>
                <li><a href="#users">Управление учетными записями</a></li>
                <li><a href="#backup">Резервное копирование</a></li>
                <li><a href="#stats">Статистика</a></li>
            </ul>
        </nav>

        <div id="add-user" class="form-section">
            <h2>Добавить пользователя</h2>
            <form>
                <label for="username">Логин:</label>
                <input type="login" id="username" name="username" required>

                <label for="password">Пароль:</label>
                <input type="password" id="password" name="password" required>

                <label for="role">Роль:</label>
                <select type="role"> 
                  <option value="Студент">Студент</option>
                  <option value="Преподаватель">Преподаватель</option>
                  <option value="Админ">Админ</option>
                </select>

                <label for="group">Группа:</label>
                <select type="group">`

	// Добавление в выпадающий список при создании учетной записи
	groups := adminData.Groups
	users := adminData.Users

	for i := 0; i < len(groups); i++ {
		adminHTML.GroupsForSelector += `<option value="` + groups[i].Name + `">` + groups[i].Name + "</option>"
		adminHTML.GroupsHTML += "<li><span>" + groups[i].Name + `</span><button class="delete-button">Удалить</button></li>` + "\n"
	}

	page += adminHTML.GroupsForSelector

	var usersSection string
	for i := 0; i < len(users); i++ {
		usersSection += "<li><span>" + users[i].Username + "</span>"
		if users[i].Role == "admin" {
			usersSection += `<select><option value="Админ">Админ</option>
			<option value="Преподаватель">Преподаватель</option>
			<option value="Студент">Студент</option>
		  </select>`
		} else if users[i].Role == "student" {
			usersSection += `<select><option value="Студент">Студент</option>
                      <option value="Преподаватель">Преподаватель</option>
                      <option value="Админ">Админ</option>
                    </select>`
		} else {
			usersSection += `<select><option value="Преподаватель">Преподаватель</option>
                      <option value="Студент">Студент</option>
                      <option value="Админ">Админ</option>
                    </select>`
		}

		grps := `<select>
                      <option value="` + users[i].GroupName + `">` + users[i].GroupName + `</option>`
		for j := 0; j < len(groups); j++ {
			if groups[j].Name != users[i].GroupName {
				grps += `<option value="` + groups[j].Name + `">` + groups[j].Name + `</option>`
			}
		}
		grps += "</select>"

		usersSection += grps
		usersSection += `<button class="delete-button">Удалить</button>
                </li>`
	}

	page += `</select>

                <button type="submit">Добавить пользователя</button>
            </form>
        </div>

        <div id="add-group" class="groups-add-section">
            <h2>Добавить группу</h2>
            <form>
                <label for="groupname">Название группы:</label>
                <input type="groupname" id="groupname" name="groupname" required>
                <button type="addgroup">Добавить группу</button>
            </form>
        </div>

        <div id="groups" class="groups-section">
            <h2>Управление группами</h2>
            <ul class="groups-list">
                `
	page += adminHTML.GroupsHTML
	page += `
            </ul>
        </div>

        <div id="users" class="users-section">
            <h2>Управление учетными записями</h2>
            <ul class="user-list">`

	// users section

	page += usersSection

	page += `</ul>
        </div>

        <div id="backup" class="backup-section">
            <h2>Резервное копирование</h2>
            <p>Нажмите на кнопку, чтобы получить архив с резервной копией системы.</p>
            <button>Создать резервную копию</button>
        </div>

        <div id="stats" class="stats-section">
            <h2>Статистика сайта</h2>
            <ul>
                <li>Всего посещений: {{.ВсегоПосещений}}</li>
                <li>Самая популярная страница: {{.СамаяПопулярнаяСтраница}}</li>
            </ul>
        </div>
    </div>
    <script src="../static/js/admin.js"></script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.New("admin").Parse(page)
	if err != nil {
		slog.Info("Не удалось получить шаблон страницы профиля")
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, stat)
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

	slog.Info(profileData.Role)
	if profileData.Role == "student" {
		profileData.Role = "Студент"
	} else if profileData.Role == "teacher" {
		profileData.Role = "Преподаватель"
	} else {
		profileData.Role = "Админ"
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

// Добавление пользователей
func handleAddUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

}

// Удаление пользователей
func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}
}

// Добавление групп
func handleAddGroup(w http.ResponseWriter, r *http.Request) {

}

// Удаление групп
func handleDeleteGroup(w http.ResponseWriter, r *http.Request) {

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

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// API
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/verify", handleVerifyToken)
	http.HandleFunc("/api/refreshtoken", handleRefreshToken)
	http.HandleFunc("/api/getprofiledata", getProfileData)
	http.HandleFunc("/api/getadminpaneldata", getAdminPanelData)

	http.HandleFunc("/api/admin/adduser", handleAddUser)
	http.HandleFunc("/api/admin/deleteuser", handleDeleteUser)
	http.HandleFunc("api/admin/addgroup", handleAddGroup)
	http.HandleFunc("api/admin/deletegroup", handleDeleteGroup)

	http.HandleFunc("api/admin/changeusergroup", handleChangeUserGroup)
	http.HandleFunc("api/admin/changeuserrole", handleChangeUserRole)

	go func() {
		<-ctx.Done()
		s.Shutdown(ctx)
	}()

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
