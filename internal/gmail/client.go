package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"
	"time"

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

	raw, err := buildMIME(fromEmail, to, subject, body, pdfName, pdfData)
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

func buildMIME(from, to, subject, body, pdfName string, pdfData []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	headers := textproto.MIMEHeader{}
	headers.Set("To", to)
	headers.Set("From", from)
	headers.Set("Subject", mime.QEncoding.Encode("utf-8", subject))
	headers.Set("MIME-Version", "1.0")
	headers.Set("Content-Type", "multipart/mixed; boundary="+writer.Boundary())
	headers.Set("Date", time.Now().Format(time.RFC1123Z))

	for k, v := range headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, "")))
	}
	buf.WriteString("\r\n")

	textPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=utf-8"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := textPart.Write([]byte(body)); err != nil {
		return nil, err
	}

	attachPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"application/pdf"},
		"Content-Disposition":       {fmt.Sprintf(`attachment; filename="%s"`, pdfName)},
		"Content-Transfer-Encoding": {"base64"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := attachPart.Write(encodeBase64Lines(pdfData)); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeBase64Lines(data []byte) []byte {
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(encoded, data)

	var out bytes.Buffer
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		out.Write(encoded[i:end])
		out.WriteString("\r\n")
	}
	return out.Bytes()
}
