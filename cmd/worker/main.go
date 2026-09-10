package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/applyflow/applyflow/internal/config"
	"github.com/applyflow/applyflow/internal/gmail"
	"github.com/applyflow/applyflow/internal/migrate"
	"github.com/applyflow/applyflow/internal/queue"
	"github.com/applyflow/applyflow/internal/repository"
	"github.com/applyflow/applyflow/internal/service"
	"github.com/applyflow/applyflow/internal/smtp"
	"github.com/applyflow/applyflow/internal/storage"
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

	sqsClient, err := queue.NewSQSClient(ctx, cfg.AWSRegion, cfg.AWSEndpointURL, cfg.SQSQueueURL)
	if err != nil {
		log.Fatalf("sqs: %v", err)
	}

	gmailSender := gmail.NewSender(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)

	var smtpSender *smtp.Sender
	if cfg.SMTPEnabled() {
		slog.Info("smtp_configured", "host", cfg.SMTPHost, "port", cfg.SMTPPort, "user", cfg.SMTPUser, "from", cfg.SMTPFrom)
		smtpSender = smtp.NewSender(smtp.Config{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			User:     cfg.SMTPUser,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
		})
	} else {
		slog.Info("smtp_not_configured")
	}

	worker := service.NewEmailWorkerService(
		pool,
		repository.NewOutreachRepository(pool),
		repository.NewCampaignRepository(pool),
		repository.NewUserRepository(pool),
		repository.NewTemplateRepository(pool),
		repository.NewResumeRepository(pool),
		repository.NewOAuthRepository(pool),
		s3Client,
		gmailSender,
		smtpSender,
	)

	workerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		slog.Info("worker started")
		for {
			select {
			case <-workerCtx.Done():
				return
			default:
				msgs, err := sqsClient.Receive(workerCtx)
				if err != nil {
					if workerCtx.Err() != nil {
						return
					}
					slog.Error("receive messages", "error", err)
					continue
				}
				for _, msg := range msgs {
					if err := worker.Process(workerCtx, msg.OutreachID); err != nil {
						slog.Error("process message", "outreach_id", msg.OutreachID, "error", err)
						continue
					}
					if err := sqsClient.Delete(workerCtx, msg.ReceiptHandle); err != nil {
						slog.Error("delete message", "error", err)
					}
				}
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
}
