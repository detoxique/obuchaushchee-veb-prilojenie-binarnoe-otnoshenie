package models

import (
	"encoding/json"
	"time"
)

// ДTO для создания теста
type CreateTestRequest struct {
	CourseID  int                     `json:"course_id"`
	Title     string                  `json:"title" validate:"required,min=3,max=255"`
	Duration  int                     `json:"duration" validate:"min=0"` // в секундах
	Attempts  int                     `json:"attempts" validate:"min=0"`
	EndDate   time.Time               `json:"end_date"`
	Questions []CreateQuestionRequest `json:"questions"`
}

// ДTO для создания вопроса
type CreateQuestionRequest struct {
	QuestionText string                      `json:"question_text" validate:"required,min=3"`
	QuestionType string                      `json:"question_type" validate:"required,oneof=single_choice multiple_choice text matching"`
	Points       int                         `json:"points" validate:"min=0"`
	Position     int                         `json:"position" validate:"min=0"`
	Options      []CreateAnswerOptionRequest `json:"options,omitempty" validate:"dive"`
}

// ДTO для создания варианта ответа
type CreateAnswerOptionRequest struct {
	OptionText string `json:"option_text" validate:"required,min=1"`
	IsCorrect  bool   `json:"is_correct"`
	Position   int    `json:"position" validate:"min=0"`
}

type UserAnswer struct {
	ID           int             `json:"id"`
	AttemptID    int             `json:"attempt_id"`
	QuestionID   int             `json:"question_id"`
	AnswerData   json.RawMessage `json:"answer_data"`
	PointsEarned int             `json:"points_earned"`
}
