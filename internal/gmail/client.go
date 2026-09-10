package gmail

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/applyflow/applyflow/internal/mail"
	"github.com/applyflow/applyflow/internal/model"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type Sender struct {
	clientID     string
	clientSecret string
	redirectURL  string
}

func NewSender(clientID, clientSecret, redirectURL string) *Sender {
	return &Sender{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
}

func (s *Sender) OAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.clientID,
		ClientSecret: s.clientSecret,
		RedirectURL:  s.redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.send",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}
}

func (s *Sender) RefreshToken(ctx context.Context, cred *model.OAuthCredential) (*oauth2.Token, error) {
	cfg := s.OAuthConfig()
	token := &oauth2.Token{
		AccessToken:  cred.AccessToken,
		RefreshToken: cred.RefreshToken,
		Expiry:       cred.ExpiresAt,
	}
	ts := cfg.TokenSource(ctx, token)
	newToken, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}
	return newToken, nil
}

func (s *Sender) Send(ctx context.Context, cred *model.OAuthCredential, fromEmail, to, subject, body string, pdfName string, pdfData []byte) (string, error) {
	token, err := s.RefreshToken(ctx, cred)
	if err != nil {
		return "", err
	}

	cred.AccessToken = token.AccessToken
	cred.ExpiresAt = token.Expiry

	client := s.OAuthConfig().Client(ctx, token)
	svc, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", fmt.Errorf("create gmail service: %w", err)
	}

	raw, err := mail.BuildMIME(fromEmail, to, subject, body, pdfName, pdfData)
	if err != nil {
		return "", err
	}

	msg := &gmail.Message{Raw: base64.URLEncoding.EncodeToString(raw)}
	sent, err := svc.Users.Messages.Send("me", msg).Do()
	if err != nil {
		return "", fmt.Errorf("send email: %w", err)
	}
	return sent.Id, nil
}
