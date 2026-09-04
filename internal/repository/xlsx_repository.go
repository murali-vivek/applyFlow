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

type XLSXRepository struct {
	pool *pgxpool.Pool
}

func NewXLSXRepository(pool *pgxpool.Pool) *XLSXRepository {
	return &XLSXRepository{pool: pool}
}

func (r *XLSXRepository) Create(ctx context.Context, f *model.XLSXFile) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO xlsx_files (user_id, file_name, storage_path)
		VALUES ($1, $2, $3)
		RETURNING id, uploaded_at`,
		f.UserID, f.FileName, f.StoragePath,
	).Scan(&f.ID, &f.UploadedAt)
}

func (r *XLSXRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*model.XLSXFile, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, file_name, storage_path, uploaded_at
		FROM xlsx_files WHERE id = $1 AND user_id = $2`, id, userID)
	return scanXLSX(row)
}

func scanXLSX(row pgx.Row) (*model.XLSXFile, error) {
	var f model.XLSXFile
	err := row.Scan(&f.ID, &f.UserID, &f.FileName, &f.StoragePath, &f.UploadedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan xlsx: %w", err)
	}
	return &f, nil
}
