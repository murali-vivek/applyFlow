package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/applyflow/applyflow/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ResumeRepository struct {
	pool *pgxpool.Pool
}

func NewResumeRepository(pool *pgxpool.Pool) *ResumeRepository {
	return &ResumeRepository{pool: pool}
}

func (r *ResumeRepository) GetByUser(ctx context.Context, userID uuid.UUID) (*model.Resume, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, name, s3_path, uploaded_at
		FROM resumes WHERE user_id = $1`, userID)
	return scanResume(row)
}

func (r *ResumeRepository) Upsert(ctx context.Context, resume *model.Resume) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO resumes (user_id, name, s3_path)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET name = EXCLUDED.name, s3_path = EXCLUDED.s3_path, uploaded_at = NOW()`,
		resume.UserID, resume.Name, resume.S3Path)
	if err != nil {
		return fmt.Errorf("upsert resume: %w", err)
	}
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, name, s3_path, uploaded_at FROM resumes WHERE user_id = $1`, resume.UserID)
	got, err := scanResume(row)
	if err != nil {
		return err
	}
	*resume = *got
	return nil
}

func scanResume(row pgx.Row) (*model.Resume, error) {
	var res model.Resume
	err := row.Scan(&res.ID, &res.UserID, &res.Name, &res.S3Path, &res.UploadedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan resume: %w", err)
	}
	return &res, nil
}
