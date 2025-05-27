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
	slog.Info("Сервер запущен. Порт: 9293")

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
	r.HandleFunc("/teachercourses", handlers.ServeTeacherCoursesPage)
	r.HandleFunc("/notifications", handlers.ServeNotificationsPage)
	r.HandleFunc("/course/{name}", handlers.ServeCoursePage)
	r.HandleFunc("/view/{name}", handlers.ServeViewPage)
	r.HandleFunc("/trainer", handlers.ServeTrainerPage)
	r.HandleFunc("/test/create/{id}", handlers.ServeCreateTestPage)

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

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
	r.HandleFunc("/api/uploadfile", handlers.HandleUploadFile)
	r.HandleFunc("/api/createcourse", handlers.HandleCreateCourse)
	r.HandleFunc("/api/deletecourse", handlers.HandleDeleteCourse)

	r.HandleFunc("/api/admin/adduser", handlers.HandleAddUser)
	r.HandleFunc("/api/admin/deleteuser", handlers.HandleDeleteUser)
	r.HandleFunc("/api/admin/addgroup", handlers.HandleAddGroup)
	r.HandleFunc("/api/admin/deletegroup", handlers.HandleDeleteGroup)

	r.HandleFunc("/api/admin/changeusergroup", handlers.HandleChangeUserGroup)
	r.HandleFunc("/api/admin/changeuserrole", handlers.HandleChangeUserRole)

	// API tests-service
	r.HandleFunc("/api/tests", handlers.CreateTest)
	r.HandleFunc("/api/tests/test/{id}", handlers.GetTest)
	r.HandleFunc("/api/tests/startattempt", handlers.StartAttempt)

	r.HandleFunc("/api/attempts/answer", handlers.SubmitAnswer)
	r.HandleFunc("/api/attempts/finish", handlers.FinishAttempt)

	http.Handle("/", r)

	if err := http.ListenAndServe(":9293", r); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
