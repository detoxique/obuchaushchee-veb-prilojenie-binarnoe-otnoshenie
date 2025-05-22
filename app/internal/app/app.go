package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/detoxique/obuchaushchee-veb-prilojenie-binarnoe-otnoshenie/app/internal/handlers"
	"github.com/detoxique/obuchaushchee-veb-prilojenie-binarnoe-otnoshenie/app/internal/models"
)

func Run(ctx context.Context) error {
	slog.Info("Сервер запущен. Порт: 8080")

	r := mux.NewRouter()
	// Загрузка статистики

	// Чтение файла
	fileData, err := os.ReadFile("stats.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(fileData, &models.Stats)
	if err != nil {
		panic(err)
	}

	// HTML
	r.HandleFunc("/", handlers.ServeLoginPage)
	r.HandleFunc("/profile", handlers.ServeProfilePage)
	r.HandleFunc("/marks", handlers.ServeMarksPage)
	r.HandleFunc("/admin", handlers.ServeAdminPage)
	r.HandleFunc("/courses", handlers.ServeCoursesPage)
	r.HandleFunc("/notifications", handlers.ServeNotificationsPage)
	r.HandleFunc("/createtest", handlers.ServeCreateTestPage)

	r.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// API
	r.HandleFunc("/api/login", handlers.HandleLogin)
	r.HandleFunc("/api/verify", handlers.HandleVerifyToken)
	r.HandleFunc("/api/verifyadmin", handlers.HandleVerifyAdmin)
	r.HandleFunc("/api/verifyteacher", handlers.HandleVerifyTeacher)
	r.HandleFunc("/api/refreshtoken", handlers.HandleRefreshToken)
	r.HandleFunc("/api/getprofiledata", handlers.GetProfileData)
	r.HandleFunc("/api/getteacherprofiledata", handlers.GetTeacherProfileData)
	r.HandleFunc("/api/getadminpaneldata", handlers.GetAdminPanelData)
	r.HandleFunc("/api/getteachercoursesdata", handlers.GetTeacherCoursesData)
	r.HandleFunc("/api/getcoursesdata", handlers.GetCoursesData)
	r.HandleFunc("/api/getteachermarksdata", handlers.GetTeacherMarksData)
	r.HandleFunc("/api/getmarksdata", handlers.GetMarksData)
	r.HandleFunc("/api/gettestsdata", handlers.GetTestsData)
	r.HandleFunc("/api/gettest", handlers.GetTest)
	r.HandleFunc("/api/upload", handlers.HandleUploadFile)

	r.HandleFunc("/api/admin/adduser", handlers.HandleAddUser)
	r.HandleFunc("/api/admin/deleteuser", handlers.HandleDeleteUser)
	r.HandleFunc("/api/admin/addgroup", handlers.HandleAddGroup)
	r.HandleFunc("/api/admin/deletegroup", handlers.HandleDeleteGroup)

	r.HandleFunc("/api/admin/changeusergroup", handlers.HandleChangeUserGroup)
	r.HandleFunc("/api/admin/changeuserrole", handlers.HandleChangeUserRole)

	// API tests-service
	r.HandleFunc("/api/tests", handlers.CreateTest)
	r.HandleFunc("/api/tests/{id}", handlers.GetTest)
	r.HandleFunc("/api/tests/startattempt", handlers.StartAttempt)

	r.HandleFunc("/api/attempts/answer", handlers.SubmitAnswer)
	r.HandleFunc("/api/attempts/finish", handlers.FinishAttempt)

	if err := http.ListenAndServe(":8080", r); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
