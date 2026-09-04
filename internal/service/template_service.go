package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/applyflow/applyflow/internal/model"
	"github.com/applyflow/applyflow/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxTemplates = 5

var (
	ErrMaxTemplates      = errors.New("maximum 5 templates allowed")
	ErrMissingCompanyVar = errors.New("template must include {{company_name}}")
	ErrCannotDeleteDefault = errors.New("cannot delete the only default template")
)

type TemplateService struct {
	pool      *pgxpool.Pool
	templates *repository.TemplateRepository
}

func NewTemplateService(pool *pgxpool.Pool, templates *repository.TemplateRepository) *TemplateService {
	return &TemplateService{pool: pool, templates: templates}
}

func (s *TemplateService) List(ctx context.Context, userID uuid.UUID) ([]model.Template, error) {
	return s.templates.ListByUser(ctx, userID)
}

func (s *TemplateService) Create(ctx context.Context, userID uuid.UUID, name, subject, body string, isDefault bool) (*model.Template, error) {
	if err := validateTemplate(name, subject, body); err != nil {
		return nil, err
	}

	count, err := s.templates.CountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= maxTemplates {
		return nil, ErrMaxTemplates
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if isDefault {
		if err := s.templates.UnsetDefault(ctx, tx, userID); err != nil {
			return nil, err
		}
	}

	t := &model.Template{
		UserID:    userID,
		Name:      name,
		Subject:   subject,
		Body:      body,
		Variables: extractVariables(subject, body),
		IsDefault: isDefault || count == 0,
	}
	if err := s.templates.Create(ctx, tx, t); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TemplateService) Update(ctx context.Context, userID, id uuid.UUID, name, subject, body string, isDefault bool) (*model.Template, error) {
	if err := validateTemplate(name, subject, body); err != nil {
		return nil, err
	}

	existing, err := s.templates.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if isDefault {
		if err := s.templates.UnsetDefault(ctx, tx, userID); err != nil {
			return nil, err
		}
	}

	existing.Name = name
	existing.Subject = subject
	existing.Body = body
	existing.Variables = extractVariables(subject, body)
	existing.IsDefault = isDefault

	if err := s.templates.Update(ctx, tx, existing); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *TemplateService) Delete(ctx context.Context, userID, id uuid.UUID) error {
	templates, err := s.templates.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	if len(templates) <= 1 {
		return ErrCannotDeleteDefault
	}

	target, err := s.templates.GetByID(ctx, id, userID)
	if err != nil {
		return err
	}
	if target.IsDefault {
		return ErrCannotDeleteDefault
	}
	return s.templates.Delete(ctx, id, userID)
}

func validateTemplate(name, subject, body string) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(subject) == "" || strings.TrimSpace(body) == "" {
		return fmt.Errorf("name, subject, and body are required")
	}
	combined := subject + body
	if !strings.Contains(combined, "{{company_name}}") {
		return ErrMissingCompanyVar
	}
	return nil
}

func extractVariables(subject, body string) []string {
	text := subject + body
	vars := []string{}
	for _, v := range []string{"company_name", "role", "user_name"} {
		if strings.Contains(text, "{{"+v+"}}") {
			vars = append(vars, v)
		}
	}
	return vars
}
