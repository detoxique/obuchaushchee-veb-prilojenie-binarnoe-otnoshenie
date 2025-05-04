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
	"strings"
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

type Statistics struct {
	ПосещенияПрофль      int64 `json:"ПосещенияПрофль"`
	ПосещенияАдминПанель int64 `json:"ПосещенияАдминПанель"`
	ПосещенияОценки      int64 `json:"ПосещенияОценки"`
	ПосещенияКурсы       int64 `json:"ПосещенияКурсы"`
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

	tmpl, err := template.ParseFiles("templates/admin.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)

	UpdateStats()
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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl, err := template.ParseFiles("templates/admin.html")
		if err != nil {
			slog.Info("Не удалось получить шаблон страницы профиля")
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, nil)
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

	go func() {
		<-ctx.Done()
		s.Shutdown(ctx)
	}()

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
