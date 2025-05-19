package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"tests-service/internal/models"
	"tests-service/internal/service"

	"github.com/go-chi/chi/v5"
)

type TestHandler struct {
	service *service.TestService
}

func NewTestHandler(s *service.TestService) *TestHandler {
	return &TestHandler{service: s}
}

func (h *TestHandler) GetTest(w http.ResponseWriter, r *http.Request) {
	testID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid test ID", http.StatusBadRequest)
		return
	}

	test, questions, err := h.service.GetTest(r.Context(), testID)
	if err != nil {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *TestHandler) StartAttempt(w http.ResponseWriter, r *http.Request) {
	infoStart := struct {
		Id     int `json:"id"`
		UserID int `json:"user_id"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&infoStart)
	if err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	attempt, err := h.service.StartAttempt(r.Context(), infoStart.UserID, infoStart.Id)
	if err != nil {
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
		UserId int               `json:"user_id"`
		Answer models.UserAnswer `json:"answer"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&answerData)
	if err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	if err := h.service.SubmitAnswer(r.Context(), answerData.UserId, &answerData.Answer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *TestHandler) FinishAttempt(w http.ResponseWriter, r *http.Request) {
	answerData := struct {
		UserId    int `json:"user_id"`
		AttemptId int `json:"attempt_id"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&answerData)
	if err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	attempt, err := h.service.FinishAttempt(r.Context(), answerData.UserId, answerData.AttemptId)
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
