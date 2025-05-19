package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/detoxique/obuchaushchee-veb-prilojenie-binarnoe-otnoshenie/app/internal/handlers"
	"github.com/detoxique/obuchaushchee-veb-prilojenie-binarnoe-otnoshenie/app/internal/models"
)

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

	err = json.Unmarshal(fileData, &models.Stats)
	if err != nil {
		panic(err)
	}

	// HTML
	http.HandleFunc("/", handlers.ServeLoginPage)
	http.HandleFunc("/profile", handlers.ServeProfilePage)
	http.HandleFunc("/marks", handlers.ServeMarksPage)
	http.HandleFunc("/admin", handlers.ServeAdminPage)
	http.HandleFunc("/courses", handlers.ServeCoursesPage)
	http.HandleFunc("/notifications", handlers.ServeNotificationsPage)
	http.HandleFunc("/createtest", handlers.ServeCreateTestPage)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// API
	http.HandleFunc("/api/login", handlers.HandleLogin)
	http.HandleFunc("/api/verify", handlers.HandleVerifyToken)
	http.HandleFunc("/api/verifyadmin", handlers.HandleVerifyAdmin)
	http.HandleFunc("/api/verifyteacher", handlers.HandleVerifyTeacher)
	http.HandleFunc("/api/refreshtoken", handlers.HandleRefreshToken)
	http.HandleFunc("/api/getprofiledata", handlers.GetProfileData)
	http.HandleFunc("/api/getteacherprofiledata", handlers.GetTeacherProfileData)
	http.HandleFunc("/api/getadminpaneldata", handlers.GetAdminPanelData)
	http.HandleFunc("/api/getteachercoursesdata", handlers.GetTeacherCoursesData)
	http.HandleFunc("/api/getcoursesdata", handlers.GetCoursesData)
	http.HandleFunc("/api/getteachermarksdata", handlers.GetTeacherMarksData)
	http.HandleFunc("/api/getmarksdata", handlers.GetMarksData)
	http.HandleFunc("/api/gettestsdata", handlers.GetTestsData)
	http.HandleFunc("/api/gettest", handlers.GetTest)
	http.HandleFunc("/api/upload", handlers.HandleUploadFile)

	http.HandleFunc("/api/admin/adduser", handlers.HandleAddUser)
	http.HandleFunc("/api/admin/deleteuser", handlers.HandleDeleteUser)
	http.HandleFunc("/api/admin/addgroup", handlers.HandleAddGroup)
	http.HandleFunc("/api/admin/deletegroup", handlers.HandleDeleteGroup)

	http.HandleFunc("/api/admin/changeusergroup", handlers.HandleChangeUserGroup)
	http.HandleFunc("/api/admin/changeuserrole", handlers.HandleChangeUserRole)

	// API tests-service
	http.HandleFunc("/api/tests", handlers.CreateTest)
	http.HandleFunc("/api/tests/get", handlers.GetTest)
	http.HandleFunc("/api/tests/startattempt", handlers.StartAttempt)

	http.HandleFunc("/api/attempts/answer", handlers.SubmitAnswer)
	http.HandleFunc("/api/attempts/finish", handlers.FinishAttempt)

	go func() {
		<-ctx.Done()
		s.Shutdown(ctx)
	}()

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
