package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/applyflow/applyflow/internal/gmail"
	"github.com/applyflow/applyflow/internal/model"
	"github.com/applyflow/applyflow/internal/render"
	"github.com/applyflow/applyflow/internal/repository"
	"github.com/applyflow/applyflow/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailWorkerService struct {
	pool      *pgxpool.Pool
	outreach  *repository.OutreachRepository
	campaigns *repository.CampaignRepository
	users     *repository.UserRepository
	templates *repository.TemplateRepository
	resumes   *repository.ResumeRepository
	oauth     *repository.OAuthRepository
	s3        *storage.S3Client
	gmail     *gmail.Sender
}

func NewEmailWorkerService(
	pool *pgxpool.Pool,
	outreach *repository.OutreachRepository,
	campaigns *repository.CampaignRepository,
	users *repository.UserRepository,
	templates *repository.TemplateRepository,
	resumes *repository.ResumeRepository,
	oauth *repository.OAuthRepository,
	s3 *storage.S3Client,
	gmailSender *gmail.Sender,
) *EmailWorkerService {
	return &EmailWorkerService{
		pool: pool, outreach: outreach, campaigns: campaigns, users: users,
		templates: templates, resumes: resumes, oauth: oauth, s3: s3, gmail: gmailSender,
	}
}

func (w *EmailWorkerService) Process(ctx context.Context, outreachID uuid.UUID) error {
	outreach, err := w.outreach.GetByID(ctx, outreachID)
	if err != nil {
		return err
	}

	switch outreach.Status {
	case model.OutreachStatusSent, model.OutreachStatusFailed, model.OutreachStatusCancelled:
		return nil
	}

	campaign, err := w.campaigns.GetByIDInternal(ctx, outreach.CampaignID)
	if err != nil {
		return err
	}

	if campaign.Status == model.CampaignStatusCancelled {
		_ = w.outreach.MarkCancelled(ctx, outreachID)
		return nil
	}

	claimed, err := w.outreach.MarkProcessingIfQueued(ctx, outreachID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	user, err := w.users.GetByID(ctx, outreach.UserID)
	if err != nil {
		return w.failOutreach(ctx, outreach, campaign, fmt.Sprintf("load user: %v", err))
	}

	tmpl, err := w.templates.GetByID(ctx, campaign.TemplateID, outreach.UserID)
	if err != nil {
		return w.failOutreach(ctx, outreach, campaign, fmt.Sprintf("load template: %v", err))
	}

	resume, err := w.resumes.GetByUser(ctx, outreach.UserID)
	if err != nil {
		return w.failOutreach(ctx, outreach, campaign, fmt.Sprintf("load resume: %v", err))
	}

	pdfData, err := w.s3.Download(ctx, resume.S3Path)
	if err != nil {
		return w.retryableFail(ctx, outreach, fmt.Sprintf("download resume: %v", err))
	}

	cred, err := w.oauth.GetByUserID(ctx, outreach.UserID)
	if err != nil {
		return w.failOutreach(ctx, outreach, campaign, "OAuth credentials not found")
	}

	subject := render.Template(tmpl.Subject, user.Name, outreach.CompanyName, outreach.Role)
	body := render.Template(tmpl.Body, user.Name, outreach.CompanyName, outreach.Role)

	slog.Info("email_send_started", "outreach_id", outreachID, "from", user.Email, "to", outreach.RecipientEmail)

	msgID, err := w.gmail.Send(ctx, cred, user.Email, outreach.RecipientEmail, subject, body, resume.Name, pdfData)
	if err != nil {
		return w.retryableFail(ctx, outreach, fmt.Sprintf("send email: %v", err))
	}

	if cred.AccessToken != "" {
		_ = w.oauth.UpdateTokens(ctx, cred.UserID, cred.AccessToken, cred.ExpiresAt)
	}

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	updated, err := w.outreach.MarkSent(ctx, tx, outreachID, msgID)
	if err != nil {
		return err
	}
	if updated {
		if err := w.campaigns.IncrementSent(ctx, tx, campaign.ID); err != nil {
			return err
		}
		if err := w.campaigns.TryComplete(ctx, tx, campaign.ID); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	slog.Info("email_sent", "outreach_id", outreachID, "from", user.Email, "to", outreach.RecipientEmail, "gmail_message_id", msgID)
	return nil
}

func (w *EmailWorkerService) failOutreach(ctx context.Context, outreach *model.Outreach, campaign *model.Campaign, errMsg string) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	updated, err := w.outreach.MarkFailed(ctx, tx, outreach.ID, errMsg)
	if err != nil {
		return err
	}
	if updated {
		if err := w.campaigns.IncrementFailed(ctx, tx, campaign.ID); err != nil {
			return err
		}
		if err := w.campaigns.TryComplete(ctx, tx, campaign.ID); err != nil {
			return err
		}
	}
	slog.Error("email_failed", "outreach_id", outreach.ID, "error", errMsg)
	return tx.Commit(ctx)
}

func (w *EmailWorkerService) retryableFail(ctx context.Context, outreach *model.Outreach, errMsg string) error {
	slog.Warn("email_retryable_failure", "outreach_id", outreach.ID, "error", errMsg)
	return fmt.Errorf("%s", errMsg)
}
