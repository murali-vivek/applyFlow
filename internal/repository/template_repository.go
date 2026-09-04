package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/applyflow/applyflow/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TemplateRepository struct {
	pool *pgxpool.Pool
}

func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

func (r *TemplateRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Template, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, name, subject, body, variables, is_default, created_at, modified_at
		FROM templates WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	templates := make([]model.Template, 0)
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *t)
	}
	return templates, rows.Err()
}

func (r *TemplateRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*model.Template, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, name, subject, body, variables, is_default, created_at, modified_at
		FROM templates WHERE id = $1 AND user_id = $2`, id, userID)
	return scanTemplate(row)
}

func (r *TemplateRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM templates WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

func (r *TemplateRepository) Create(ctx context.Context, tx pgx.Tx, t *model.Template) error {
	vars, err := json.Marshal(t.Variables)
	if err != nil {
		return fmt.Errorf("marshal variables: %w", err)
	}
	return tx.QueryRow(ctx, `
		INSERT INTO templates (user_id, name, subject, body, variables, is_default)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, modified_at`,
		t.UserID, t.Name, t.Subject, t.Body, vars, t.IsDefault,
	).Scan(&t.ID, &t.CreatedAt, &t.ModifiedAt)
}

func (r *TemplateRepository) Update(ctx context.Context, tx pgx.Tx, t *model.Template) error {
	vars, err := json.Marshal(t.Variables)
	if err != nil {
		return fmt.Errorf("marshal variables: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE templates SET name = $1, subject = $2, body = $3, variables = $4, is_default = $5, modified_at = NOW()
		WHERE id = $6 AND user_id = $7`,
		t.Name, t.Subject, t.Body, vars, t.IsDefault, t.ID, t.UserID)
	return err
}

func (r *TemplateRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM templates WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TemplateRepository) UnsetDefault(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	_, err := tx.Exec(ctx, `UPDATE templates SET is_default = FALSE, modified_at = NOW() WHERE user_id = $1 AND is_default = TRUE`, userID)
	return err
}

func (r *TemplateRepository) CreateDefault(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.Template, error) {
	t := &model.Template{
		UserID:    userID,
		Name:      "Default",
		Subject:   "Application for {{role}} at {{company_name}}",
		Body:      "Hi,\n\nI am reaching out regarding {{role}} opportunities at {{company_name}}.\n\nI have attached my resume for your consideration.\n\nRegards,\n{{user_name}}",
		Variables: []string{"company_name", "role", "user_name"},
		IsDefault: true,
	}
	if err := r.Create(ctx, tx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func scanTemplate(row pgx.Row) (*model.Template, error) {
	var t model.Template
	var varsJSON []byte
	err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.Subject, &t.Body, &varsJSON, &t.IsDefault, &t.CreatedAt, &t.ModifiedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan template: %w", err)
	}
	if len(varsJSON) > 0 {
		_ = json.Unmarshal(varsJSON, &t.Variables)
	}
	return &t, nil
}
