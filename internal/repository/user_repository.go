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

var ErrNotFound = errors.New("not found")

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, email, created_at, modified_at
		FROM users WHERE email = $1`, email)
	return scanUser(row)
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, email, created_at, modified_at
		FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (r *UserRepository) Create(ctx context.Context, name, email string) (*model.User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (name, email)
		VALUES ($1, $2)
		RETURNING id, name, email, created_at, modified_at`, name, email)
	return scanUser(row)
}

func scanUser(row pgx.Row) (*model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.ModifiedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}
