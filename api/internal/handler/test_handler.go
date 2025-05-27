package handler

import (
	"api/internal/models"
	"api/internal/service"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type TestHandler struct {
	service *service.TestService
}

func NewTestHandler(s *service.TestService) *TestHandler {
	return &TestHandler{service: s}
}

func (h *TestHandler) GetTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]

	log.Println("Получаем тест с id: " + testID)

	test, questions, err := h.service.GetTest(r.Context(), testID)
	if err != nil {
		log.Println("Error getting test " + err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response := struct {
		Test      *models.Test      `json:"test"`
		Questions []models.Question `json:"questions"`
	}{
		Test:      test,
		Questions: questions,
	}

	j, err := json.Marshal(response)
	if err != nil {
		log.Println("Error getting test " + err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	log.Println(string(j))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *TestHandler) StartAttempt(w http.ResponseWriter, r *http.Request) {
	infoStart := struct {
		Id    string `json:"id"`
		Token string `json:"token"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&infoStart)
	if err != nil {
		log.Println("Некорректный запрос " + err.Error())
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	username := service.GetUsernameGromToken(infoStart.Token)

	var UserID int
	err = h.service.Repo.Db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&UserID)
	if err != nil {
		log.Println("Некорректный запрос " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	testId, err := strconv.Atoi(infoStart.Id)
	if err != nil {
		log.Println("Некорректный запрос(test_handler.StartAttempt) " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Попытка начать тест с id: " + strconv.Itoa(testId))

	attempt, err := h.service.StartAttempt(r.Context(), UserID, testId)
	if err != nil {
		log.Println("Некорректный запрос " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(attempt)
}

func (h *TestHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	// attemptID, err := strconv.Atoi(chi.URLParam(r, "attempt_id"))
	// if err != nil {
	// 	http.Error(w, "Invalid attempt ID", http.StatusBadRequest)
	// 	return
	// }

	// userID, err := strconv.Atoi(chi.URLParam(r, "iduser"))
	// if err != nil {
	// 	http.Error(w, "Invalid test ID", http.StatusBadRequest)
	// 	return
	// }

	answerData := struct {
		Answer models.UserAnswer `json:"answer"`
		Token  string            `json:"token"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&answerData)
	if err != nil {
		log.Println("Некорректный запрос " + err.Error())
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	username := service.GetUsernameGromToken(answerData.Token)

	var UserID int
	err = h.service.Repo.Db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&UserID)
	if err != nil {
		log.Println("Некорректный запрос " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.service.SubmitAnswer(r.Context(), UserID, &answerData.Answer); err != nil {
		log.Println("Некорректный запрос " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TestHandler) FinishAttempt(w http.ResponseWriter, r *http.Request) {
	answerData := struct {
		Token  string `json:"token"`
		TestID int    `json:"test_id"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&answerData)
	if err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	username := service.GetUsernameGromToken(answerData.Token)

	var UserID int
	err = h.service.Repo.Db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var AttemptId int
	err = h.service.Repo.Db.QueryRow("SELECT id FROM test_attempts WHERE user_id = $1 AND status = 'in_progress'", UserID).Scan(&AttemptId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	attempt, err := h.service.FinishAttempt(r.Context(), UserID, AttemptId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(attempt)
}

func (h *TestHandler) CreateTest(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация (можно использовать github.com/go-playground/validator)
	// if err := validator.New().Struct(req); err != nil {
	//     http.Error(w, err.Error(), http.StatusBadRequest)
	//     return
	// }

	test, err := h.service.CreateTest(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(test)
}
