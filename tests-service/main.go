package main

import (
	"database/sql"
	"log"
	"net/http"
	"tests-service/internal/handler"
	"tests-service/internal/repository"
	"tests-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

func main() {
	// Подключение к базе данных
	db, err := sql.Open("postgres", "postgres://postgres:123@localhost/portaldb?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Инициализация слоев приложения
	testRepo := repository.NewTestRepository(db)
	testService := service.NewTestService(testRepo)
	testHandler := handler.NewTestHandler(testService)

	// Настройка маршрутов
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Route("/api/tests", func(r chi.Router) {
		r.Post("/", testHandler.CreateTest)
		r.Get("/{id}", testHandler.GetTest)
		r.Post("/{id}/{iduser}/attempts", testHandler.StartAttempt)
	})

	r.Route("/api/attempts", func(r chi.Router) {
		r.Post("/{attempt_id}/{iduser}/answers", testHandler.SubmitAnswer)
		r.Post("/{attempt_id}/{iduser}/finish", testHandler.FinishAttempt)
	})

	// Запуск сервера
	log.Println("Starting server on :1310")
	log.Fatal(http.ListenAndServe(":1310", r))
}
