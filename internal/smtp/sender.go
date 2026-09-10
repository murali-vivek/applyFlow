package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/applyflow/applyflow/internal/mail"
	"github.com/google/uuid"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func (c Config) Enabled() bool {
	return c.Host != "" && c.User != "" && c.Password != "" && c.From != ""
}

type Sender struct {
	cfg Config
}

func NewSender(cfg Config) *Sender {
	return &Sender{cfg: cfg}
}

func (s *Sender) FromAddress() string {
	return s.cfg.From
}

func (s *Sender) Send(ctx context.Context, to, subject, body, pdfName string, pdfData []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	raw, err := mail.BuildMIME(s.cfg.From, to, subject, body, pdfName, pdfData)
	if err != nil {
		return "", err
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)

	if s.cfg.Port == 465 {
		if err := sendTLS(addr, s.cfg.Host, auth, s.cfg.From, []string{to}, raw); err != nil {
			return "", fmt.Errorf("smtp send: %w", err)
		}
	} else {
		if err := smtp.SendMail(addr, auth, s.cfg.From, []string{to}, raw); err != nil {
			return "", fmt.Errorf("smtp send: %w", err)
		}
	}

	return uuid.NewString(), nil
}

func sendTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func ParseHostDefault(host string) string {
	h := strings.TrimSpace(host)
	if h == "" {
		return "smtp.zoho.com"
	}
	return h
}
