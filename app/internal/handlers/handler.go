package handlers

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/detoxique/obuchaushchee-veb-prilojenie-binarnoe-otnoshenie/app/internal/models"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Резервное копирование
func HandleBackup(w http.ResponseWriter, r *http.Request) {
	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "backup_*")
	if err != nil {
		slog.Info("Ошибка " + err.Error())
		http.Error(w, "Failed to create temp dir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	// Копируем папку static/pdf
	pdfBackupPath := filepath.Join(tempDir, "pdf_backup")
	if err := copyDir("static/pdf", pdfBackupPath); err != nil {
		slog.Info("Ошибка " + err.Error())
		http.Error(w, "PDF backup failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Создаем дамп базы данных
	dumpPath := filepath.Join(tempDir, "db_dump.sql")
	if err := createDBDump(dumpPath); err != nil {
		slog.Info("Ошибка " + err.Error())
		http.Error(w, "Database dump failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Создаем ZIP-архив в памяти
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=backup_"+time.Now().Format("20060102-150405")+".zip")

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Добавляем файлы в архив
	if err := addFilesToZip(zipWriter, tempDir, ""); err != nil {
		slog.Info("Ошибка " + err.Error())
		http.Error(w, "ZIP creation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Info("Ошибка " + err.Error())
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			slog.Info("Ошибка " + err.Error())
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, os.ModePerm)
		}

		return copyFile(path, targetPath)
	})
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		slog.Info("Ошибка " + err.Error())
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		slog.Info("Ошибка " + err.Error())
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func createDBDump(dumpPath string) error {
	// Параметры подключения к БД (лучше вынести в конфиг/переменные окружения)
	connStr := "postgres://postgres:123@localhost/portaldb?sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("DB connection failed: %w", err)
	}
	defer db.Close()

	file, err := os.Create(dumpPath)
	if err != nil {
		return fmt.Errorf("file creation failed: %w", err)
	}
	defer file.Close()

	// 1. Экспорт схемы таблиц
	if err := exportSchema(db, file); err != nil {
		return fmt.Errorf("schema export failed: %w", err)
	}

	// 2. Экспорт данных
	if err := exportData(db, file); err != nil {
		return fmt.Errorf("data export failed: %w", err)
	}

	return nil
}

func exportSchema(db *sql.DB, file *os.File) error {
	// Получаем список таблиц
	tables, err := getTables(db)
	if err != nil {
		return err
	}

	// Для каждой таблицы получаем DDL
	for _, table := range tables {
		var ddl string
		err := db.QueryRow(
			`SELECT 'CREATE TABLE ' || $1 || ' (' || 
            string_agg(column_definition, ', ') || ');'
            FROM (
                SELECT 
                    column_name || ' ' || data_type || 
                    CASE WHEN character_maximum_length IS NOT NULL 
                         THEN '(' || character_maximum_length || ')' 
                         ELSE '' 
                    END || 
                    CASE WHEN is_nullable = 'NO' THEN ' NOT NULL' ELSE '' END AS column_definition
                FROM information_schema.columns
                WHERE table_name = $1
                ORDER BY ordinal_position
            ) AS columns;`,
			table,
		).Scan(&ddl)

		if err != nil {
			return fmt.Errorf("failed to get DDL for %s: %w", table, err)
		}

		file.WriteString(ddl + "\n\n")
	}
	return nil
}

func getTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(
		`SELECT table_name 
         FROM information_schema.tables 
         WHERE table_schema = 'public' 
           AND table_type = 'BASE TABLE'`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}

func exportData(db *sql.DB, file *os.File) error {
	tables, err := getTables(db)
	if err != nil {
		return err
	}

	for _, table := range tables {
		// Получаем данные таблицы
		rows, err := db.Query(fmt.Sprintf(`SELECT * FROM "%s"`, table))
		if err != nil {
			return fmt.Errorf("failed to query table %s: %w", table, err)
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to get columns for %s: %w", table, err)
		}

		// Для каждой строки формируем INSERT
		for rows.Next() {
			values := make([]interface{}, len(columns))
			pointers := make([]interface{}, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}

			if err := rows.Scan(pointers...); err != nil {
				return fmt.Errorf("row scan failed: %w", err)
			}

			var valueStrings []string
			for _, v := range values {
				switch value := v.(type) {
				case nil:
					valueStrings = append(valueStrings, "NULL")
				case string:
					valueStrings = append(valueStrings, "'"+escapeSQLString(value)+"'")
				case []byte:
					valueStrings = append(valueStrings, "'"+escapeSQLString(string(value))+"'")
				default:
					valueStrings = append(valueStrings, fmt.Sprintf("%v", value))
				}
			}

			insert := fmt.Sprintf(
				"INSERT INTO \"%s\" VALUES (%s);\n",
				table,
				strings.Join(valueStrings, ", "),
			)
			file.WriteString(insert)
		}
		file.WriteString("\n")
	}
	return nil
}

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func addFilesToZip(zipWriter *zip.Writer, basePath, baseInZip string) error {
	return filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Info("Ошибка " + err.Error())
			return err
		}

		relPath, _ := filepath.Rel(basePath, path)
		if relPath == "." {
			return nil
		}

		zipPath := filepath.Join(baseInZip, relPath)

		if info.IsDir() {
			_, err := zipWriter.Create(zipPath + "/")
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			slog.Info("Ошибка " + err.Error())
			return err
		}
		defer file.Close()

		zipFile, err := zipWriter.Create(zipPath)
		if err != nil {
			slog.Info("Ошибка " + err.Error())
			return err
		}

		_, err = io.Copy(zipFile, file)
		return err
	})
}

// Страница авторизации
func ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// Страница Профиля
func ServeProfilePage(w http.ResponseWriter, r *http.Request) {
	models.Stats.ПосещенияПрофль++

	tmpl, err := template.ParseFiles("templates/profile.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)

	UpdateStats()
}

// Страница Профиля
func ServeCoursesPage(w http.ResponseWriter, r *http.Request) {
	models.Stats.ПосещенияКурсы++

	tmpl, err := template.ParseFiles("templates/courses.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)

	UpdateStats()
}

func ServeTeacherCoursesPage(w http.ResponseWriter, r *http.Request) {
	models.Stats.ПосещенияКурсы++

	tmpl, err := template.ParseFiles("templates/coursesteacher.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)

	UpdateStats()
}

// Страница оценок
func ServeMarksPage(w http.ResponseWriter, r *http.Request) {
	models.Stats.ПосещенияОценки++

	tmpl, err := template.ParseFiles("templates/marks.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)

	UpdateStats()
}

// Страница авторизации
func ServeNotificationsPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/notifications.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// Страница тренажера
func ServeTrainerPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/trainer.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// Страница админ панели
func ServeAdminPage(w http.ResponseWriter, r *http.Request) {
	// TODO: Проверять, авторизован ли пользователь в учетку админа

	models.Stats.ПосещенияАдминПанель++

	var mostPopular string
	if models.Stats.ПосещенияАдминПанель > models.Stats.ПосещенияПрофль && models.Stats.ПосещенияАдминПанель > models.Stats.ПосещенияОценки && models.Stats.ПосещенияАдминПанель > models.Stats.ПосещенияКурсы {
		mostPopular = "Админ Панель"
	} else if models.Stats.ПосещенияПрофль > models.Stats.ПосещенияАдминПанель && models.Stats.ПосещенияПрофль > models.Stats.ПосещенияОценки && models.Stats.ПосещенияПрофль > models.Stats.ПосещенияКурсы {
		mostPopular = "Профиль"
	} else if models.Stats.ПосещенияОценки > models.Stats.ПосещенияПрофль && models.Stats.ПосещенияОценки > models.Stats.ПосещенияАдминПанель && models.Stats.ПосещенияОценки > models.Stats.ПосещенияКурсы {
		mostPopular = "Оценки"
	} else {
		mostPopular = "Курсы"
	}

	stat := models.StatsToView{
		ВсегоПосещений:          models.Stats.ПосещенияАдминПанель + models.Stats.ПосещенияОценки + models.Stats.ПосещенияПрофль + models.Stats.ПосещенияКурсы,
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

func ServeCreateTestPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&id)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/getcoursenamebyid", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера(get course name): " + strconv.Itoa(resp.StatusCode))
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	var name string

	err = json.NewDecoder(resp.Body).Decode(&name)
	if err != nil {
		slog.Info("Не удалось считать данные профиля" + err.Error())
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	info := struct {
		Coursename string `json:"Coursename"`
	}{Coursename: name}

	tmpl, err := template.ParseFiles("templates/test.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, info)
}

// Получение страницы профиля с данными
func GetAdminPanelData(w http.ResponseWriter, r *http.Request) {
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
	var adminData models.AdminPanelData

	err = json.NewDecoder(resp.Body).Decode(&adminData)
	if err != nil {
		slog.Info("Не удалось считать данные профиля")
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	var mostPopular string
	if models.Stats.ПосещенияАдминПанель > models.Stats.ПосещенияПрофль && models.Stats.ПосещенияАдминПанель > models.Stats.ПосещенияОценки && models.Stats.ПосещенияАдминПанель > models.Stats.ПосещенияКурсы {
		mostPopular = "Админ Панель"
	} else if models.Stats.ПосещенияПрофль > models.Stats.ПосещенияАдминПанель && models.Stats.ПосещенияПрофль > models.Stats.ПосещенияОценки && models.Stats.ПосещенияПрофль > models.Stats.ПосещенияКурсы {
		mostPopular = "Профиль"
	} else if models.Stats.ПосещенияОценки > models.Stats.ПосещенияПрофль && models.Stats.ПосещенияОценки > models.Stats.ПосещенияАдминПанель && models.Stats.ПосещенияОценки > models.Stats.ПосещенияКурсы {
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

	data := models.ServeAdminPanelData{
		Groups:         template.HTML(sel),
		GroupsTable:    template.HTML(groupsTable),
		UsersTable:     template.HTML(usersTable),
		ВсегоПосещений: models.Stats.ПосещенияАдминПанель + models.Stats.ПосещенияОценки + models.Stats.ПосещенияПрофль + models.Stats.ПосещенияКурсы,
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
func GetProfileData(w http.ResponseWriter, r *http.Request) {
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
	var profileData models.ProfilePageData

	err = json.NewDecoder(respData.Body).Decode(&profileData)
	if err != nil {
		slog.Info("Не удалось считать данные профиля " + err.Error())
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	var tests, courses_cards string
	for i := 0; i < len(profileData.Courses); i++ {
		// Самый поздний тест
		var date time.Time
		for j := 0; j < len(profileData.Courses[i].Tests); j++ {
			if profileData.Courses[i].Tests[j].EndDate.Compare(date) > 0 {
				date = profileData.Courses[i].Tests[j].EndDate
			}
		}
		if len(profileData.Courses[i].Tests) > 0 {
			if len(profileData.Courses[i].Tests)%10 == 0 {
				tests += `<li><a href="#">` + profileData.Courses[i].Name + `</a><br><h4>` + strconv.Itoa(len(profileData.Courses[i].Tests)) + ` тестов до ` + date.Format("02.01.2006") + `!</h4></li>`
			} else if len(profileData.Courses[i].Tests)%10 == 1 {
				tests += `<li><a href="#">` + profileData.Courses[i].Name + `</a><br><h4>` + strconv.Itoa(len(profileData.Courses[i].Tests)) + ` тест до ` + date.Format("02.01.2006") + `!</h4></li>`
			} else if len(profileData.Courses[i].Tests)%10 > 1 && len(profileData.Courses[i].Tests)%10 < 5 {
				tests += `<li><a href="#">` + profileData.Courses[i].Name + `</a><br><h4>` + strconv.Itoa(len(profileData.Courses[i].Tests)) + ` теста до ` + date.Format("02.01.2006") + `!</h4></li>`
			} else {
				tests += `<li><a href="#">` + profileData.Courses[i].Name + `</a><br><h4>` + strconv.Itoa(len(profileData.Courses[i].Tests)) + ` тестов до ` + date.Format("02.01.2006") + `!</h4></li>`
			}
		} else {
			tests += `<li><a href="#">` + profileData.Courses[i].Name + `</a><br><h4>Тесты пока не загружены.</h4></li>`
		}

		if i > 0 {
			courses_cards += `<div class="card-2">
                        <div class="card" style="width: 18rem;">
                            <div class="card-body">
                              <h5 class="card-title">` + profileData.Courses[i].Name + `</h5>
                              <h6 class="card-subtitle mb-2 text-body-secondary">Тестов: ` + strconv.Itoa(len(profileData.Courses[i].Tests)) + `</h6>
                              <p class="card-text"></p><a href="/course/` + profileData.Courses[i].Name + `" class="card-link">Перейти к курсу.</a>
                            </div>
                        </div>
                    </div>`
		} else {
			courses_cards += `<div class="card-1">
                        <div class="card" style="width: 18rem;">
                            <div class="card-body">
                              <h5 class="card-title">` + profileData.Courses[i].Name + `</h5>
                              <h6 class="card-subtitle mb-2 text-body-secondary">Тестов: ` + strconv.Itoa(len(profileData.Courses[i].Tests)) + `</h6>
                              <p class="card-text"></p><a href="/course/` + profileData.Courses[i].Name + `" class="card-link">Перейти к курсу.</a>
                            </div>
                        </div>
                    </div>`
		}

	}

	profileHTML := struct {
		Username     template.HTML `json:"Username"`
		Group        template.HTML `json:"Group"`
		TestsList    template.HTML `json:"TestsList"`
		CoursesCards template.HTML `json:"CoursesCards"`
	}{
		Username:     template.HTML(profileData.Username),
		Group:        template.HTML(profileData.Group),
		TestsList:    template.HTML(tests),
		CoursesCards: template.HTML(courses_cards),
	}

	// Отправление страницы пользователю
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles("templates/profile.html")
	if err != nil {
		slog.Info("Не удалось получить шаблон страницы профиля")
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, profileHTML)
}

// Получение страницы профиля с данными
func GetTeacherProfileData(w http.ResponseWriter, r *http.Request) {
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
	var profileData models.ProfilePageData

	err = json.NewDecoder(respData.Body).Decode(&profileData)
	if err != nil {
		slog.Info("Не удалось считать данные профиля")
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	var coursesHTML string
	for i := 0; i < len(profileData.Courses); i++ {
		coursesHTML += `<li><a href="#">` + profileData.Courses[i].Name + `</a><br><h4>Материалов: ` + strconv.Itoa(len(profileData.Courses[i].Files)) + ` Тестов: ` + strconv.Itoa(len(profileData.Courses[i].Tests)) + `</h4></li>`
	}

	data := struct {
		Username template.HTML `json:"Username"`
		Courses  template.HTML `json:"Courses"`
	}{
		Username: template.HTML(profileData.Username),
		Courses:  template.HTML(coursesHTML),
	}

	// Отправление страницы пользователю
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles("templates/profileteacher.html")
	if err != nil {
		slog.Info("Не удалось получить шаблон страницы профиля " + err.Error())
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

func GetTeacherCoursesData(w http.ResponseWriter, r *http.Request) {
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
	var coursesData models.TeacherCoursesPageData

	err = json.NewDecoder(respData.Body).Decode(&coursesData)
	if err != nil {
		slog.Info("Не удалось считать данные профиля")
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(coursesData)
}

func GetCoursesData(w http.ResponseWriter, r *http.Request) {
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
	var coursesData models.CoursesPageData

	err = json.NewDecoder(respData.Body).Decode(&coursesData)
	if err != nil {
		slog.Info("Не удалось считать данные профиля" + err.Error())
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	slog.Info("Количество курсов: " + strconv.Itoa(len(coursesData.Courses)))

	coursesHTML := `<ul class="courses-list">`

	for i := 0; i < len(coursesData.Courses); i++ {
		if len(coursesData.Courses[i].Tests) > 0 {
			coursesHTML += `<li><a href="/course/` + coursesData.Courses[i].Name + `">` + coursesData.Courses[i].Name + `</a><br><h4>Тест до ` + coursesData.Courses[i].Tests[0].EndDate.Format("02.01.2006") + `!</h4></li>`
		} else {
			coursesHTML += `<li><a href="/course/` + coursesData.Courses[i].Name + `">` + coursesData.Courses[i].Name + `</a><br><h4></h4></li>`
		}

	}

	coursesHTML += `</ul>`

	var data models.CoursesPageServeData

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

func GetTeacherMarksData(w http.ResponseWriter, r *http.Request) {
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

func GetMarksData(w http.ResponseWriter, r *http.Request) {
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

func GetTestsData(w http.ResponseWriter, r *http.Request) {
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
	//slog.Info(string(body))

	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func ServeCoursePage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// Отправка запроса на другой сервер
	resp, err := http.Get("http://localhost:1337/api/getcoursedata/" + name)
	if err != nil {
		slog.Info("Ошибка получения данных тестов http")
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var pageData models.ServeCoursePage

	err = json.NewDecoder(resp.Body).Decode(&pageData)
	if err != nil {
		slog.Info("Не удалось считать данные профиля" + err.Error())
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	var filesHTML string
	for i := 0; i < len(pageData.Course.Files); i++ {
		filesHTML += `<li><a href="/view/` + pageData.Course.Files[i].Name + `">` + pageData.Course.Files[i].Name + `</a><br><h4>Загружено: ` + pageData.Course.Files[i].UploadDate.Format("02.01.2006") + `</h4></li>`
	}

	var testsHTML string
	for i := 0; i < len(pageData.Course.Tests); i++ {
		testsHTML += `<li><a href="#">` + pageData.Course.Tests[i].Title + `</a><br><h4>Должен быть выполнен до: ` + pageData.Course.Tests[i].EndDate.Format("02.01.2006") + `</h4></li>`
	}

	slog.Info("Количество тестов: " + strconv.Itoa(len(pageData.Course.Tests)))

	data := struct {
		CourseName template.HTML `json:"CourseName"`
		Files      template.HTML `json:"Files"`
		Tests      template.HTML `json:"Tests"`
	}{
		CourseName: template.HTML(pageData.Course.Name),
		Files:      template.HTML(filesHTML),
		Tests:      template.HTML(testsHTML),
	}

	tmpl, err := template.ParseFiles("templates/course.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func ServeViewPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// Отправка запроса на другой сервер
	resp, err := http.Get("http://localhost:1337/api/getviewdata/" + name)
	if err != nil {
		slog.Info("Ошибка получения данных тестов http")
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var filename string

	err = json.NewDecoder(resp.Body).Decode(&filename)
	if err != nil {
		slog.Info("Не удалось считать данные профиля" + err.Error())
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	data := struct {
		Name     template.HTML `json:"Name"`
		Filename template.HTML `json:"Filename"`
	}{
		Name:     template.HTML(name),
		Filename: template.HTML(filename),
	}

	tmpl, err := template.ParseFiles("templates/view.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

// Авторизация
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var loginData models.LoginData

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

func HandleVerifyToken(w http.ResponseWriter, r *http.Request) {
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

func HandleVerifyAdmin(w http.ResponseWriter, r *http.Request) {
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

func HandleVerifyTeacher(w http.ResponseWriter, r *http.Request) {
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

func HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
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

func HandleUploadFile(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на добавление файла")
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Ограничиваем размер файла
	//r.Body = http.MaxBytesReader(w, r.Body, 128)
	// body, _ := io.ReadAll(r.Body)
	// slog.Info(string(body))

	// // Отправка запроса на другой сервер
	// resp, err := http.Post("http://localhost:1337/api/uploadfile", "multipart/form-data", bytes.NewBuffer(body))
	// if err != nil {
	// 	http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
	// 	return
	// }
	// defer resp.Body.Close()

	// // Чтение ответа от другого сервера
	// w.Header().Set("Content-Type", "application/json")
	// if resp.StatusCode != http.StatusOK {
	// 	// Перенаправление ошибки от другого сервера
	// 	body, _ := io.ReadAll(resp.Body)
	// 	w.WriteHeader(resp.StatusCode)
	// 	w.Write(body)
	// 	return
	// }
	// w.Write(body)
	targetURL, _ := url.Parse("http://localhost:1337/api/uploadfile")

	proxyReq := &http.Request{
		Method: r.Method,
		URL:    targetURL,
		Header: r.Header.Clone(), // Клонируем заголовки
		Body:   r.Body,           // Используем оригинальное тело
	}

	// 3. Копируем важные стандартные заголовки
	if r.Host != "" {
		proxyReq.Header.Set("Host", r.Host)
	}
	if r.ContentLength > 0 {
		proxyReq.ContentLength = r.ContentLength
	}

	// 4. Отправляем запрос к целевому серверу
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Proxy error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 5. Копируем заголовки ответа
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 6. Устанавливаем статус код
	w.WriteHeader(resp.StatusCode)

	// 7. Копируем тело ответа
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Println("Error copying response body:", err)
	}
}

func HandleCreateCourse(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на создание курса")
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/createcourse", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	body, _ = io.ReadAll(resp.Body)
	w.Write(body)
}

func HandleDeleteCourse(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на удаление курса")
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/deletecourse", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	body, _ = io.ReadAll(resp.Body)
	w.Write(body)
}

// Добавление пользователей
func HandleAddUser(w http.ResponseWriter, r *http.Request) {
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

	var userData models.UserData

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
func HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
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

	var userData models.DeleteUserData

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
	var delUser models.DeleteUser
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
func HandleAddGroup(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на добавление группы в БД")
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var groupData models.GroupData

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
func HandleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на удаление группы из БД")
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var data models.DeleteGroupData

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
func HandleChangeUserGroup(w http.ResponseWriter, r *http.Request) {

}

// Изменение роли пользователя
func HandleChangeUserRole(w http.ResponseWriter, r *http.Request) {

}

// Тесты
func CreateTest(w http.ResponseWriter, r *http.Request) {
	slog.Info("Получен запрос на создание теста")
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var testrequest models.CreateTestRequest

	err := json.NewDecoder(r.Body).Decode(&testrequest)
	if err != nil {
		slog.Info("Не удалось считать данные для входа " + err.Error())
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	// Подготовка запроса к другому серверу
	body, err := json.Marshal(&testrequest.Token)
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

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка сервера авторизации(не препод)")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Токен подтвержден
	// Подготовка запроса к другому серверу
	body, err = json.Marshal(testrequest)
	if err != nil {
		slog.Info("Ошибка преобразования в JSON")
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	// Отправка запроса на другой сервер
	resp, err = http.Post("http://localhost:1337/api/tests/", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации(отправка запроса)", http.StatusInternalServerError)
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

func GetTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	slog.Info("Получен запрос на получение данных теста с ID " + id)
	if r.Method != "GET" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Отправка запроса на другой сервер
	resp, err := http.Get("http://localhost:1337/api/tests/test/" + id)
	if err != nil {
		slog.Info("Ошибка при получении теста")
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

func StartAttempt(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/tests/attempts", "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Ошибка сервера авторизации", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Невозможно начать попытку.")
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

func SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)

	slog.Info(string(body))

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/attempts/answers", "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Info("Ошибка отправки ответа")
		http.Error(w, "Ошибка отправки ответа", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа от другого сервера
	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode != http.StatusOK {
		// Перенаправление ошибки от другого сервера
		slog.Info("Ошибка отправки ответа")
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Успешный ответ
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		slog.Info("Ошибка чтения ответа")
		http.Error(w, "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

func FinishAttempt(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		slog.Info("Метод не разрешен")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}

	// Отправка запроса на другой сервер
	resp, err := http.Post("http://localhost:1337/api/attempts/finish", "application/json", bytes.NewBuffer(body))
	if err != nil {
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
	jsonData, err := json.MarshalIndent(models.Stats, "", "  ")
	if err != nil {
		panic(err)
	}

	// Запись в файл "stats.json"
	err = os.WriteFile("stats.json", jsonData, 0644) // Права 0644 (rw-r--r--)
	if err != nil {
		panic(err)
	}
}
