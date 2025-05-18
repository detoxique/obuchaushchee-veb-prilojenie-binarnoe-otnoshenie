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
	testID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid test ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(chi.URLParam(r, "iduser"))
	if err != nil {
		http.Error(w, "Invalid test ID", http.StatusBadRequest)
		return
	}

	attempt, err := h.service.StartAttempt(r.Context(), userID, testID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(attempt)
}

func (h *TestHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	attemptID, err := strconv.Atoi(chi.URLParam(r, "attempt_id"))
	if err != nil {
		http.Error(w, "Invalid attempt ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(chi.URLParam(r, "iduser"))
	if err != nil {
		http.Error(w, "Invalid test ID", http.StatusBadRequest)
		return
	}

	var answer models.UserAnswer
	if err := json.NewDecoder(r.Body).Decode(&answer); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	answer.AttemptID = attemptID

	if err := h.service.SubmitAnswer(r.Context(), userID, &answer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *TestHandler) FinishAttempt(w http.ResponseWriter, r *http.Request) {
	attemptID, err := strconv.Atoi(chi.URLParam(r, "attempt_id"))
	if err != nil {
		http.Error(w, "Invalid attempt ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(chi.URLParam(r, "iduser"))
	if err != nil {
		http.Error(w, "Invalid test ID", http.StatusBadRequest)
		return
	}

	attempt, err := h.service.FinishAttempt(r.Context(), userID, attemptID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(attempt)
}
