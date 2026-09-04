package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/applyflow/applyflow/internal/auth"
	"github.com/applyflow/applyflow/internal/config"
	"github.com/applyflow/applyflow/internal/gmail"
	"github.com/applyflow/applyflow/internal/handler"
	"github.com/applyflow/applyflow/internal/middleware"
	"github.com/applyflow/applyflow/internal/migrate"
	"github.com/applyflow/applyflow/internal/queue"
	"github.com/applyflow/applyflow/internal/repository"
	"github.com/applyflow/applyflow/internal/service"
	"github.com/applyflow/applyflow/internal/storage"
	"github.com/google/uuid"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := repository.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := migrate.Run(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	s3Client, err := storage.NewS3Client(ctx, cfg.AWSRegion, cfg.AWSEndpointURL, cfg.S3Bucket)
	if err != nil {
		log.Fatalf("s3: %v", err)
	}
	if err := s3Client.EnsureBucket(ctx); err != nil {
		slog.Warn("ensure s3 bucket", "error", err)
	}

	sqsClient, err := queue.NewSQSClient(ctx, cfg.AWSRegion, cfg.AWSEndpointURL, cfg.SQSQueueURL)
	if err != nil {
		log.Fatalf("sqs: %v", err)
	}

	jwtMgr := auth.NewJWTManager(cfg.JWTSecret)
	gmailSender := gmail.NewSender(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)

	userRepo := repository.NewUserRepository(pool)
	templateRepo := repository.NewTemplateRepository(pool)
	resumeRepo := repository.NewResumeRepository(pool)
	xlsxRepo := repository.NewXLSXRepository(pool)
	campaignRepo := repository.NewCampaignRepository(pool)
	outreachRepo := repository.NewOutreachRepository(pool)
	oauthRepo := repository.NewOAuthRepository(pool)

	authSvc := service.NewAuthService(pool, userRepo, templateRepo, oauthRepo, gmailSender, jwtMgr, cfg.FrontendURL)
	templateSvc := service.NewTemplateService(pool, templateRepo)
	resumeSvc := service.NewResumeService(resumeRepo, s3Client)
	xlsxSvc := service.NewXLSXService(xlsxRepo, s3Client)
	campaignSvc := service.NewCampaignService(pool, campaignRepo, outreachRepo, xlsxSvc, templateRepo, resumeRepo, oauthRepo)
	schedulerSvc := service.NewSchedulerService(outreachRepo, campaignRepo, sqsClient, cfg.SchedulerInterval, cfg.StaleQueuedAfter)

	authHandler := handler.NewAuthHandler(authSvc)
	templateHandler := handler.NewTemplateHandler(templateSvc)
	resumeHandler := handler.NewResumeHandler(resumeSvc)
	xlsxHandler := handler.NewXLSXHandler(xlsxSvc)
	campaignHandler := handler.NewCampaignHandler(campaignSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("GET /auth/google", authHandler.GoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", authHandler.GoogleCallback)
	mux.HandleFunc("GET /auth/logout", authHandler.Logout)

	authMW := middleware.Auth(jwtMgr)
	mux.Handle("GET /auth/me", authMW(http.HandlerFunc(authHandler.Me)))

	mux.Handle("GET /templates", authMW(http.HandlerFunc(templateHandler.List)))
	mux.Handle("POST /templates", authMW(http.HandlerFunc(templateHandler.Create)))
	mux.Handle("PUT /templates/{id}", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		templateHandler.Update(w, r, id)
	})))
	mux.Handle("DELETE /templates/{id}", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		templateHandler.Delete(w, r, id)
	})))

	mux.Handle("GET /resume", authMW(http.HandlerFunc(resumeHandler.Get)))
	mux.Handle("POST /resume", authMW(http.HandlerFunc(resumeHandler.Upload)))
	mux.Handle("POST /files/xlsx", authMW(http.HandlerFunc(xlsxHandler.Upload)))

	mux.Handle("GET /campaigns", authMW(http.HandlerFunc(campaignHandler.List)))
	mux.Handle("POST /campaigns", authMW(http.HandlerFunc(campaignHandler.Create)))
	mux.Handle("GET /campaigns/{id}", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		campaignHandler.Get(w, r, id)
	})))
	mux.Handle("POST /campaigns/{id}/cancel", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		campaignHandler.Cancel(w, r, id)
	})))

	root := middleware.CORS(cfg.FrontendURL)(mux)

	schedCtx, schedCancel := context.WithCancel(context.Background())
	defer schedCancel()
	go schedulerSvc.Run(schedCtx)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           root,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("api server starting", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	schedCancel()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
