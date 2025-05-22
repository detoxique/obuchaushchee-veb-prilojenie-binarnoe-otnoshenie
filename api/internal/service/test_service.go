package service

import (
	"api/internal/models"
	"api/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"
)

type TestService struct {
	Repo *repository.TestRepository
}

func NewTestService(repo *repository.TestRepository) *TestService {
	return &TestService{Repo: repo}
}

func (s *TestService) GetTest(ctx context.Context, testID string) (*models.Test, []models.Question, error) {
	test, err := s.Repo.GetTestByID(ctx, testID)
	if err != nil {
		log.Println("Ошибка получения теста по ID теста(файл test_service метод GetTest) " + err.Error())
		return nil, nil, err
	}

	questions, err := s.Repo.GetTestQuestions(ctx, testID)
	if err != nil {
		log.Println("Ошибка получения вопросов по ID теста(файл test_service метод GetTest) " + err.Error())
		return nil, nil, err
	}

	// Загружаем варианты ответов для каждого вопроса
	for i := range questions {
		if questions[i].QuestionType == "single_choice" || questions[i].QuestionType == "multiple_choice" {
			options, err := s.Repo.GetQuestionOptions(ctx, questions[i].ID)
			if err != nil {
				log.Println("Ошибка получения вариантов ответа по ID вопроса(файл test_service метод GetTest) " + err.Error())
				return nil, nil, err
			}
			questions[i].Options = options
		} else if questions[i].QuestionType == "text_answer" {
			option := models.AnswerOption{
				ID:         -1,
				QuestionID: -1,
				OptionText: "",
				Position:   -1,
			}
			questions[i].Options = []models.AnswerOption{option}
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

	if err := s.Repo.CreateAttempt(ctx, attempt); err != nil {
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
	return s.Repo.SaveAnswer(ctx, answer)
}

func (s *TestService) evaluateAnswer(ctx context.Context, questionID int, answerData json.RawMessage) (int, error) {
	// Получаем вопрос и его тип
	questions, err := s.Repo.GetTestQuestions(ctx, strconv.Itoa(questionID))
	if err != nil || len(questions) == 0 {
		return 0, fmt.Errorf("question not found")
	}
	question := questions[0]

	// Получаем правильные ответы (если нужно)
	var options []models.AnswerOption
	if question.QuestionType == "single_choice" || question.QuestionType == "multiple_choice" {
		options, err = s.Repo.GetQuestionOptions(ctx, questionID)
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

	case "text_answer":
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
	if err := s.Repo.CompleteAttempt(ctx, attemptID, totalScore); err != nil {
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

func (s *TestService) CreateTest(ctx context.Context, req *models.CreateTestRequest) (*models.Test, error) {
	// Валидация (используйте github.com/go-playground/validator)
	// if err := validator.New().Struct(req); err != nil {
	//     return nil, err
	// }

	// Проверка вопросов и ответов
	for _, q := range req.Questions {
		if (q.QuestionType == "single_choice" || q.QuestionType == "multiple_choice") && len(q.Options) == 0 {
			return nil, fmt.Errorf("question type %s requires options", q.QuestionType)
		}

		// Для single_choice проверяем, что есть ровно один правильный ответ
		if q.QuestionType == "single_choice" {
			correctCount := 0
			for _, opt := range q.Options {
				if opt.IsCorrect {
					correctCount++
				}
			}
			if correctCount != 1 {
				return nil, errors.New("single_choice questions must have exactly one correct option")
			}
		}
	}

	// Создаем тест
	test := &models.Test{
		CourseID: req.CourseID,
		Title:    req.Title,
		EndDate:  req.EndDate,
		Duration: req.Duration,
		Attempts: req.Attempts,
	}

	// Начинаем транзакцию
	tx, err := s.Repo.DB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txRepo := repository.NewTxRepository(tx)

	// Сохраняем тест
	if err := txRepo.CreateTest(ctx, test); err != nil {
		return nil, fmt.Errorf("failed to create test: %w", err)
	}

	// Сохраняем вопросы и варианты ответов
	for _, qReq := range req.Questions {
		question := &models.Question{
			TestID:       test.ID,
			QuestionText: qReq.QuestionText,
			QuestionType: qReq.QuestionType,
			Points:       qReq.Points,
			Position:     qReq.Position,
		}

		if err := txRepo.CreateQuestion(ctx, question); err != nil {
			return nil, fmt.Errorf("failed to create question: %w", err)
		}

		// Сохраняем варианты ответов (если есть)
		for _, optReq := range qReq.Options {
			option := &models.AnswerOption{
				QuestionID: question.ID,
				OptionText: optReq.OptionText,
				IsCorrect:  optReq.IsCorrect,
				Position:   optReq.Position,
			}

			if err := txRepo.CreateAnswerOption(ctx, option); err != nil {
				return nil, fmt.Errorf("failed to create answer option: %w", err)
			}
		}
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return test, nil
}
