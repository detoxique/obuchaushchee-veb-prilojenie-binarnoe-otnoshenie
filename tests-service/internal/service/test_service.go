package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"tests-service/internal/models"
	"tests-service/internal/repository"
	"time"
)

type TestService struct {
	repo *repository.TestRepository
}

func NewTestService(repo *repository.TestRepository) *TestService {
	return &TestService{repo: repo}
}

func (s *TestService) GetTest(ctx context.Context, testID int) (*models.Test, []models.Question, error) {
	test, err := s.repo.GetTestByID(ctx, testID)
	if err != nil {
		return nil, nil, err
	}

	questions, err := s.repo.GetTestQuestions(ctx, testID)
	if err != nil {
		return nil, nil, err
	}

	// Загружаем варианты ответов для каждого вопроса
	for i := range questions {
		if questions[i].QuestionType == "single_choice" || questions[i].QuestionType == "multiple_choice" {
			options, err := s.repo.GetQuestionOptions(ctx, questions[i].ID)
			if err != nil {
				return nil, nil, err
			}
			questions[i].Options = options
		}
	}

	return test, questions, nil
}

func (s *TestService) StartAttempt(ctx context.Context, userID, testID int) (*models.TestAttempt, error) {
	// Проверяем, можно ли начать тест (например, не превышено ли max_attempts)

	attempt := &models.TestAttempt{
		UserID:    userID,
		TestID:    testID,
		StartedAt: time.Now(),
		Status:    "in_progress",
	}

	if err := s.repo.CreateAttempt(ctx, attempt); err != nil {
		return nil, err
	}

	return attempt, nil
}

func (s *TestService) SubmitAnswer(ctx context.Context, userID int, answer *models.UserAnswer) error {
	// Проверяем, принадлежит ли attempt пользователю
	// Получаем вопрос и правильные ответы

	// В зависимости от типа вопроса проверяем ответ
	points, err := s.evaluateAnswer(ctx, answer.QuestionID, answer.AnswerData)
	if err != nil {
		return err
	}

	answer.PointsEarned = points
	return s.repo.SaveAnswer(ctx, answer)
}

func (s *TestService) evaluateAnswer(ctx context.Context, questionID int, answerData json.RawMessage) (int, error) {
	// Получаем вопрос и его тип
	questions, err := s.repo.GetTestQuestions(ctx, questionID)
	if err != nil || len(questions) == 0 {
		return 0, fmt.Errorf("question not found")
	}
	question := questions[0]

	// Получаем правильные ответы (если нужно)
	var options []models.AnswerOption
	if question.QuestionType == "single_choice" || question.QuestionType == "multiple_choice" {
		options, err = s.repo.GetQuestionOptions(ctx, questionID)
		if err != nil {
			return 0, err
		}
	}

	// Проверяем ответ в зависимости от типа вопроса
	switch question.QuestionType {
	case "single_choice":
		var answer struct {
			SelectedOptionID int `json:"selected_option_id"`
		}
		if err := json.Unmarshal(answerData, &answer); err != nil {
			return 0, err
		}

		for _, opt := range options {
			if opt.ID == answer.SelectedOptionID && opt.IsCorrect {
				return question.Points, nil
			}
		}
		return 0, nil

	case "multiple_choice":
		var answer struct {
			SelectedOptionIDs []int `json:"selected_option_ids"`
		}
		if err := json.Unmarshal(answerData, &answer); err != nil {
			return 0, err
		}

		correctOptions := 0
		for _, opt := range options {
			if opt.IsCorrect {
				correctOptions++
			}
		}

		userCorrect := 0
		for _, selectedID := range answer.SelectedOptionIDs {
			for _, opt := range options {
				if opt.ID == selectedID && opt.IsCorrect {
					userCorrect++
					break
				}
			}
		}

		// Начисляем пропорционально количеству правильных ответов
		if correctOptions > 0 {
			return (question.Points * userCorrect) / correctOptions, nil
		}
		return 0, nil

	case "text":
		// TODO: сложная проверка

		return 0, nil

	case "matching":
		// Аналогично для сопоставления
		return 0, nil

	default:
		return 0, errors.New("unknown question type")
	}
}

func (s *TestService) FinishAttempt(ctx context.Context, userID, attemptID int) (*models.TestAttempt, error) {
	// Проверяем, принадлежит ли attempt пользователю

	// Считаем общий балл за попытку
	totalScore, err := s.calculateAttemptScore(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	// Обновляем попытку
	if err := s.repo.CompleteAttempt(ctx, attemptID, totalScore); err != nil {
		return nil, err
	}

	// Возвращаем обновленную попытку
	// TODO: добавить метод для получения попытки
	return &models.TestAttempt{
		ID:     attemptID,
		UserID: userID,
		Score:  &totalScore,
		Status: "completed",
	}, nil
}

func (s *TestService) calculateAttemptScore(ctx context.Context, attemptID int) (int, error) {
	// TODO: запрашивать все ответы и суммировать баллы
	// Здесь упрощенный вариант
	return 0, nil
}
