package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"tests-service/internal/models"
)

type TestRepository struct {
	db *sql.DB
}

func NewTestRepository(db *sql.DB) *TestRepository {
	return &TestRepository{db: db}
}

func (r *TestRepository) GetTestByID(ctx context.Context, id int) (*models.Test, error) {
	query := `SELECT id, title, description, author_id, created_at, time_limit, 
              max_attempts, is_published, passing_score FROM tests WHERE id = $1`

	var test models.Test
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&test.ID, &test.Title, &test.Description, &test.AuthorID, &test.CreatedAt,
		&test.TimeLimit, &test.MaxAttempts, &test.IsPublished, &test.PassingScore,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("test not found")
		}
		return nil, err
	}

	return &test, nil
}

func (r *TestRepository) GetTestQuestions(ctx context.Context, testID int) ([]models.Question, error) {
	query := `SELECT id, test_id, question_text, question_type, points, position 
              FROM questions WHERE test_id = $1 ORDER BY position`

	rows, err := r.db.QueryContext(ctx, query, testID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []models.Question
	for rows.Next() {
		var q models.Question
		if err := rows.Scan(&q.ID, &q.TestID, &q.QuestionText, &q.QuestionType, &q.Points, &q.Position); err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}

	return questions, nil
}

func (r *TestRepository) GetQuestionOptions(ctx context.Context, questionID int) ([]models.AnswerOption, error) {
	query := `SELECT id, question_id, option_text, is_correct, position 
              FROM answer_options WHERE question_id = $1 ORDER BY position`

	rows, err := r.db.QueryContext(ctx, query, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []models.AnswerOption
	for rows.Next() {
		var opt models.AnswerOption
		if err := rows.Scan(&opt.ID, &opt.QuestionID, &opt.OptionText, &opt.IsCorrect, &opt.Position); err != nil {
			return nil, err
		}
		options = append(options, opt)
	}

	return options, nil
}

func (r *TestRepository) CreateAttempt(ctx context.Context, attempt *models.TestAttempt) error {
	query := `INSERT INTO test_attempts (user_id, test_id, started_at, status) 
              VALUES ($1, $2, $3, $4) RETURNING id`

	err := r.db.QueryRowContext(ctx, query,
		attempt.UserID, attempt.TestID, attempt.StartedAt, attempt.Status).Scan(&attempt.ID)
	return err
}

func (r *TestRepository) SaveAnswer(ctx context.Context, answer *models.UserAnswer) error {
	query := `INSERT INTO user_answers (attempt_id, question_id, answer_data, points_earned)
              VALUES ($1, $2, $3, $4) RETURNING id`

	err := r.db.QueryRowContext(ctx, query,
		answer.AttemptID, answer.QuestionID, answer.AnswerData, answer.PointsEarned).Scan(&answer.ID)
	return err
}

func (r *TestRepository) CompleteAttempt(ctx context.Context, attemptID int, score int) error {
	query := `UPDATE test_attempts SET finished_at = NOW(), score = $1, status = 'completed' 
              WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, score, attemptID)
	return err
}
