package models

import (
	"encoding/json"
	"time"
)

type Test struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	AuthorID     int       `json:"author_id"`
	CreatedAt    time.Time `json:"created_at"`
	TimeLimit    int       `json:"time_limit"` // в секундах
	MaxAttempts  int       `json:"max_attempts"`
	IsPublished  bool      `json:"is_published"`
	PassingScore int       `json:"passing_score"`
}

type Question struct {
	ID           int            `json:"id"`
	TestID       int            `json:"test_id"`
	QuestionText string         `json:"question_text"`
	QuestionType string         `json:"question_type"` // single_choice, multiple_choice, text, matching
	Points       int            `json:"points"`
	Position     int            `json:"position"`
	Options      []AnswerOption `json:"options,omitempty"`
}

type AnswerOption struct {
	ID         int    `json:"id"`
	QuestionID int    `json:"question_id"`
	OptionText string `json:"option_text"`
	IsCorrect  bool   `json:"is_correct"`
	Position   int    `json:"position"`
}

type TestAttempt struct {
	ID         int        `json:"id"`
	UserID     int        `json:"user_id"`
	TestID     int        `json:"test_id"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Score      *int       `json:"score,omitempty"`
	Status     string     `json:"status"` // in_progress, completed, expired
}

type UserAnswer struct {
	ID           int             `json:"id"`
	AttemptID    int             `json:"attempt_id"`
	QuestionID   int             `json:"question_id"`
	AnswerData   json.RawMessage `json:"answer_data"`
	PointsEarned int             `json:"points_earned"`
}
