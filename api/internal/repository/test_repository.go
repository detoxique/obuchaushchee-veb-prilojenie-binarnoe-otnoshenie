package repository

import (
	"api/internal/models"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type TestRepository struct {
	Db *sql.DB
}

func NewTestRepository(db *sql.DB) *TestRepository {
	return &TestRepository{Db: db}
}

func (r *TestRepository) DB() *sql.DB {
	return r.Db
}

// txRepository реализация для работы в транзакции (*sql.Tx)
type txRepository struct {
	tx *sql.Tx
}

// NewTxRepository создает репозиторий для работы в транзакции
func NewTxRepository(tx *sql.Tx) *txRepository {
	return &txRepository{tx: tx}
}

func (r *TestRepository) GetTestByID(ctx context.Context, id int) (*models.Test, error) {
	query := `SELECT id, name, upload_date, end_date, duration, 
              attempts FROM tests WHERE id = $1`

	var test models.Test
	err := r.Db.QueryRowContext(ctx, query, id).Scan(
		&test.ID, &test.Title, &test.UploadDate, &test.EndDate,
		&test.Duration, &test.Attempts,
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

	rows, err := r.Db.QueryContext(ctx, query, testID)
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

	rows, err := r.Db.QueryContext(ctx, query, questionID)
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

	err := r.Db.QueryRowContext(ctx, query,
		attempt.UserID, attempt.TestID, attempt.StartedAt, attempt.Status).Scan(&attempt.ID)
	return err
}

func (r *TestRepository) SaveAnswer(ctx context.Context, answer *models.UserAnswer) error {
	query := `INSERT INTO user_answers (attempt_id, question_id, answer_data, points_earned)
              VALUES ($1, $2, $3, $4) RETURNING id`

	err := r.Db.QueryRowContext(ctx, query,
		answer.AttemptID, answer.QuestionID, answer.AnswerData, answer.PointsEarned).Scan(&answer.ID)
	return err
}

func (r *TestRepository) CompleteAttempt(ctx context.Context, attemptID int, score int) error {
	query := `UPDATE test_attempts SET finished_at = NOW(), score = $1, status = 'completed' 
              WHERE id = $2`

	_, err := r.Db.ExecContext(ctx, query, score, attemptID)
	return err
}

func (r *TestRepository) CreateTest(ctx context.Context, test *models.Test) error {
	query := `INSERT INTO tests (title, end_date, duration, 
              attempts, id_course) 
              VALUES ($1, $2, $3, $4, $5) RETURNING id, upload_date`

	err := r.Db.QueryRowContext(ctx, query,
		test.Title, test.EndDate, test.Duration,
		test.Attempts, test.CourseID,
	).Scan(&test.ID, &test.UploadDate)

	return err
}

func (r *TestRepository) CreateQuestion(ctx context.Context, question *models.Question) error {
	query := `INSERT INTO questions (test_id, question_text, question_type, points, position)
              VALUES ($1, $2, $3, $4, $5) RETURNING id`

	err := r.Db.QueryRowContext(ctx, query,
		question.TestID, question.QuestionText, question.QuestionType,
		question.Points, question.Position,
	).Scan(&question.ID)

	return err
}

func (r *TestRepository) CreateAnswerOption(ctx context.Context, option *models.AnswerOption) error {
	query := `INSERT INTO answer_options (question_id, option_text, is_correct, position)
              VALUES ($1, $2, $3, $4) RETURNING id`

	err := r.Db.QueryRowContext(ctx, query,
		option.QuestionID, option.OptionText, option.IsCorrect, option.Position,
	).Scan(&option.ID)

	return err
}

func (r *txRepository) CreateTest(ctx context.Context, test *models.Test) error {
	query := `INSERT INTO tests (title, end_date, duration, 
              attempts) 
              VALUES ($1, $2, $3, $4) RETURNING id, upload_date`

	err := r.tx.QueryRowContext(ctx, query,
		test.Title, test.EndDate, test.Duration,
		test.Attempts,
	).Scan(&test.ID, &test.UploadDate)

	return err
}

func (r *txRepository) CreateQuestion(ctx context.Context, question *models.Question) error {
	query := `INSERT INTO questions (test_id, question_text, question_type, points, position)
              VALUES ($1, $2, $3, $4, $5) RETURNING id`

	err := r.tx.QueryRowContext(ctx, query,
		question.TestID, question.QuestionText, question.QuestionType,
		question.Points, question.Position,
	).Scan(&question.ID)

	return err
}

func (r *txRepository) CreateAnswerOption(ctx context.Context, option *models.AnswerOption) error {
	query := `INSERT INTO answer_options (question_id, option_text, is_correct, position)
              VALUES ($1, $2, $3, $4) RETURNING id`

	err := r.tx.QueryRowContext(ctx, query,
		option.QuestionID, option.OptionText, option.IsCorrect, option.Position,
	).Scan(&option.ID)

	return err
}
