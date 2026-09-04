package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	HTTPPort    string
	DatabaseURL string

	AWSRegion      string
	AWSEndpointURL string
	S3Bucket       string
	SQSQueueURL    string

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	JWTSecret   string
	FrontendURL string

	SchedulerInterval time.Duration
	StaleQueuedAfter  time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		HTTPPort:           getEnv("HTTP_PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		AWSRegion:          getEnv("AWS_REGION", "us-east-1"),
		AWSEndpointURL:     os.Getenv("AWS_ENDPOINT_URL"),
		S3Bucket:           getEnv("S3_BUCKET", "applyflow-uploads"),
		SQSQueueURL:        os.Getenv("SQS_QUEUE_URL"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5173"),
		SchedulerInterval:  15 * time.Second,
		StaleQueuedAfter:   5 * time.Minute,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if isPlaceholder(cfg.GoogleClientID) || isPlaceholder(cfg.GoogleClientSecret) {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must be set in .env (get them from Google Cloud Console → APIs & Services → Google Auth Platform → Clients)")
	}

	if v := os.Getenv("SCHEDULER_INTERVAL_SECONDS"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid SCHEDULER_INTERVAL_SECONDS: %w", err)
		}
		cfg.SchedulerInterval = time.Duration(secs) * time.Second
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func isPlaceholder(v string) bool {
	placeholders := []string{
		"your-google-client-id",
		"your-google-client-secret",
		"your-google-client-id.apps.googleusercontent.com",
		"change-me-to-a-long-random-string",
	}
	for _, p := range placeholders {
		if v == p || strings.Contains(v, "your-google") {
			return true
		}
	}
	return v == ""
}
