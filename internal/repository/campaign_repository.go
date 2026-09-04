package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/applyflow/applyflow/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CampaignRepository struct {
	pool *pgxpool.Pool
}

func NewCampaignRepository(pool *pgxpool.Pool) *CampaignRepository {
	return &CampaignRepository{pool: pool}
}

func (r *CampaignRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Campaign, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, campaign_number, xlsx_file_id, template_id, status, total, sent, failed,
		       created_at, scheduled_at, started_at, finished_at
		FROM campaigns WHERE user_id = $1 ORDER BY campaign_number DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	campaigns := make([]model.Campaign, 0)
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, err
		}
		campaigns = append(campaigns, *c)
	}
	return campaigns, rows.Err()
}

func (r *CampaignRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*model.Campaign, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, campaign_number, xlsx_file_id, template_id, status, total, sent, failed,
		       created_at, scheduled_at, started_at, finished_at
		FROM campaigns WHERE id = $1 AND user_id = $2`, id, userID)
	return scanCampaign(row)
}

func (r *CampaignRepository) GetByIDInternal(ctx context.Context, id uuid.UUID) (*model.Campaign, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, campaign_number, xlsx_file_id, template_id, status, total, sent, failed,
		       created_at, scheduled_at, started_at, finished_at
		FROM campaigns WHERE id = $1`, id)
	return scanCampaign(row)
}

func (r *CampaignRepository) NextCampaignNumber(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (int, error) {
	var num int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(campaign_number), 0) + 1 FROM campaigns WHERE user_id = $1`, userID).Scan(&num)
	return num, err
}

func (r *CampaignRepository) Create(ctx context.Context, tx pgx.Tx, c *model.Campaign) error {
	return tx.QueryRow(ctx, `
		INSERT INTO campaigns (user_id, campaign_number, xlsx_file_id, template_id, status, total, scheduled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		c.UserID, c.CampaignNumber, c.XLSXFileID, c.TemplateID, c.Status, c.Total, c.ScheduledAt,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *CampaignRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE campaigns SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (r *CampaignRepository) MarkRunning(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE campaigns SET status = $1, started_at = COALESCE(started_at, NOW())
		WHERE id = $2 AND status IN ('SCHEDULED', 'RUNNING')`,
		model.CampaignStatusRunning, id)
	return err
}

func (r *CampaignRepository) IncrementSent(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `UPDATE campaigns SET sent = sent + 1 WHERE id = $1`, id)
	return err
}

func (r *CampaignRepository) IncrementFailed(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `UPDATE campaigns SET failed = failed + 1 WHERE id = $1`, id)
	return err
}

func (r *CampaignRepository) TryComplete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		UPDATE campaigns SET status = $1, finished_at = NOW()
		WHERE id = $2 AND status NOT IN ('COMPLETED', 'CANCELLED')
		  AND sent + failed >= total AND total > 0`,
		model.CampaignStatusCompleted, id)
	return err
}

func (r *CampaignRepository) Cancel(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE campaigns SET status = $1, finished_at = NOW()
		WHERE id = $2 AND user_id = $3 AND status NOT IN ('COMPLETED', 'CANCELLED')`,
		model.CampaignStatusCancelled, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanCampaign(row pgx.Row) (*model.Campaign, error) {
	var c model.Campaign
	err := row.Scan(&c.ID, &c.UserID, &c.CampaignNumber, &c.XLSXFileID, &c.TemplateID, &c.Status,
		&c.Total, &c.Sent, &c.Failed, &c.CreatedAt, &c.ScheduledAt, &c.StartedAt, &c.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan campaign: %w", err)
	}
	return &c, nil
}

type OutreachRepository struct {
	pool *pgxpool.Pool
}

func NewOutreachRepository(pool *pgxpool.Pool) *OutreachRepository {
	return &OutreachRepository{pool: pool}
}

func (r *OutreachRepository) CreateBatch(ctx context.Context, tx pgx.Tx, items []model.Outreach) error {
	for _, o := range items {
		_, err := tx.Exec(ctx, `
			INSERT INTO outreach (user_id, campaign_id, company_name, role, recipient_email, status, scheduled_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			o.UserID, o.CampaignID, o.CompanyName, o.Role, o.RecipientEmail, o.Status, o.ScheduledAt)
		if err != nil {
			return fmt.Errorf("insert outreach: %w", err)
		}
	}
	return nil
}

func (r *OutreachRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Outreach, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, campaign_id, company_name, role, recipient_email, status,
		       scheduled_at, sent_at, failed_at, gmail_message_id, error_message, created_at
		FROM outreach WHERE id = $1`, id)
	return scanOutreach(row)
}

func (r *OutreachRepository) ClaimDue(ctx context.Context, limit int) ([]uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT o.id FROM outreach o
		JOIN campaigns c ON c.id = o.campaign_id
		WHERE o.status = $1 AND o.scheduled_at <= NOW() AND c.status != $2
		ORDER BY o.scheduled_at
		FOR UPDATE OF o SKIP LOCKED
		LIMIT $3`, model.OutreachStatusPending, model.CampaignStatusCancelled, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, id := range ids {
		_, err := tx.Exec(ctx, `UPDATE outreach SET status = $1, queued_at = NOW() WHERE id = $2`, model.OutreachStatusQueued, id)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *OutreachRepository) ResetToPending(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE outreach SET status = $1, queued_at = NULL WHERE id = $2 AND status = $3`,
		model.OutreachStatusPending, id, model.OutreachStatusQueued)
	return err
}

func (r *OutreachRepository) RecoverStaleQueued(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	tag, err := r.pool.Exec(ctx, `
		UPDATE outreach SET status = $1, queued_at = NULL
		WHERE status = $2 AND queued_at IS NOT NULL AND queued_at < $3`,
		model.OutreachStatusPending, model.OutreachStatusQueued, cutoff)
	return tag.RowsAffected(), err
}

func (r *OutreachRepository) MarkProcessingIfQueued(ctx context.Context, id uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE outreach SET status = $1 WHERE id = $2 AND status = $3`,
		model.OutreachStatusProcessing, id, model.OutreachStatusQueued)
	return tag.RowsAffected() > 0, err
}

func (r *OutreachRepository) MarkSent(ctx context.Context, tx pgx.Tx, id uuid.UUID, gmailMessageID string) (bool, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE outreach SET status = $1, sent_at = NOW(), gmail_message_id = $2
		WHERE id = $3 AND status = $4`,
		model.OutreachStatusSent, gmailMessageID, id, model.OutreachStatusProcessing)
	return tag.RowsAffected() > 0, err
}

func (r *OutreachRepository) MarkFailed(ctx context.Context, tx pgx.Tx, id uuid.UUID, errMsg string) (bool, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE outreach SET status = $1, failed_at = NOW(), error_message = $2
		WHERE id = $3 AND status = $4`,
		model.OutreachStatusFailed, errMsg, id, model.OutreachStatusProcessing)
	return tag.RowsAffected() > 0, err
}

func (r *OutreachRepository) MarkCancelled(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE outreach SET status = $1
		WHERE id = $2 AND status IN ('PENDING', 'QUEUED', 'PROCESSING')`,
		model.OutreachStatusCancelled, id)
	return err
}

func (r *OutreachRepository) CancelPendingForCampaign(ctx context.Context, campaignID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE outreach SET status = $1
		WHERE campaign_id = $2 AND status IN ('PENDING', 'QUEUED')`,
		model.OutreachStatusCancelled, campaignID)
	return err
}

func scanOutreach(row pgx.Row) (*model.Outreach, error) {
	var o model.Outreach
	err := row.Scan(&o.ID, &o.UserID, &o.CampaignID, &o.CompanyName, &o.Role, &o.RecipientEmail,
		&o.Status, &o.ScheduledAt, &o.SentAt, &o.FailedAt, &o.GmailMessageID, &o.ErrorMessage, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan outreach: %w", err)
	}
	return &o, nil
}
