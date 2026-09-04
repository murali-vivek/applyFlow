package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/applyflow/applyflow/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OAuthRepository struct {
	pool *pgxpool.Pool
}

func NewOAuthRepository(pool *pgxpool.Pool) *OAuthRepository {
	return &OAuthRepository{pool: pool}
}

func (r *OAuthRepository) Upsert(ctx context.Context, cred *model.OAuthCredential) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO oauth_credentials (user_id, provider, access_token, refresh_token, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			expires_at = EXCLUDED.expires_at,
			modified_at = NOW()
		RETURNING id, created_at, modified_at`,
		cred.UserID, cred.Provider, cred.AccessToken, cred.RefreshToken, cred.ExpiresAt,
	).Scan(&cred.ID, &cred.CreatedAt, &cred.ModifiedAt)
}

func (r *OAuthRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.OAuthCredential, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, provider, access_token, refresh_token, expires_at, created_at, modified_at
		FROM oauth_credentials WHERE user_id = $1`, userID)
	return scanOAuth(row)
}

func (r *OAuthRepository) ExistsForUser(ctx context.Context, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM oauth_credentials WHERE user_id = $1)`, userID).Scan(&exists)
	return exists, err
}

func scanOAuth(row pgx.Row) (*model.OAuthCredential, error) {
	var c model.OAuthCredential
	err := row.Scan(&c.ID, &c.UserID, &c.Provider, &c.AccessToken, &c.RefreshToken,
		&c.ExpiresAt, &c.CreatedAt, &c.ModifiedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan oauth: %w", err)
	}
	return &c, nil
}

func (r *OAuthRepository) UpdateTokens(ctx context.Context, userID uuid.UUID, accessToken string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE oauth_credentials SET access_token = $1, expires_at = $2, modified_at = NOW()
		WHERE user_id = $3`, accessToken, expiresAt, userID)
	return err
}
