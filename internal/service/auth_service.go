package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/applyflow/applyflow/internal/auth"
	"github.com/applyflow/applyflow/internal/gmail"
	"github.com/applyflow/applyflow/internal/model"
	"github.com/applyflow/applyflow/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

type AuthService struct {
	pool         *pgxpool.Pool
	users        *repository.UserRepository
	templates    *repository.TemplateRepository
	oauth        *repository.OAuthRepository
	gmail        *gmail.Sender
	jwt          *auth.JWTManager
	frontendURL  string
}

func NewAuthService(
	pool *pgxpool.Pool,
	users *repository.UserRepository,
	templates *repository.TemplateRepository,
	oauth *repository.OAuthRepository,
	gmailSender *gmail.Sender,
	jwt *auth.JWTManager,
	frontendURL string,
) *AuthService {
	return &AuthService{
		pool: pool, users: users, templates: templates, oauth: oauth,
		gmail: gmailSender, jwt: jwt, frontendURL: frontendURL,
	}
}

func (s *AuthService) AuthURL(state string) string {
	return s.gmail.OAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

func (s *AuthService) HandleCallback(ctx context.Context, code string) (string, error) {
	token, err := s.gmail.OAuthConfig().Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("exchange code: %w", err)
	}

	client := s.gmail.OAuthConfig().Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", fmt.Errorf("get userinfo: %w", err)
	}
	defer resp.Body.Close()

	var info struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := decodeJSON(resp, &info); err != nil {
		return "", err
	}
	if info.Email == "" {
		return "", errors.New("email not provided by Google")
	}

	user, err := s.users.GetByEmail(ctx, info.Email)
	if errors.Is(err, repository.ErrNotFound) {
		user, err = s.createUserWithDefaults(ctx, info.Name, info.Email)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	cred := &model.OAuthCredential{
		UserID:       user.ID,
		Provider:     "google",
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
	}
	if err := s.oauth.Upsert(ctx, cred); err != nil {
		return "", fmt.Errorf("store oauth: %w", err)
	}

	return s.jwt.Generate(user.ID, user.Email)
}

func (s *AuthService) createUserWithDefaults(ctx context.Context, name, email string) (*model.User, error) {
	user, err := s.users.Create(ctx, name, email)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := s.templates.CreateDefault(ctx, tx, user.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) GetOrCreateUser(ctx context.Context, name, email string) (*model.User, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		return s.createUserWithDefaults(ctx, name, email)
	}
	return user, err
}

func (s *AuthService) FrontendURL() string { return s.frontendURL }
