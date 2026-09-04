package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/applyflow/applyflow/internal/model"
	"github.com/applyflow/applyflow/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrResumeRequired = errors.New("resume is required before starting a campaign")
	ErrOAuthRequired  = errors.New("Google account not connected")
)

const outreachInterval = 3 * time.Minute

type CampaignService struct {
	pool      *pgxpool.Pool
	campaigns *repository.CampaignRepository
	outreach  *repository.OutreachRepository
	xlsx      *XLSXService
	templates *repository.TemplateRepository
	resumes   *repository.ResumeRepository
	oauth     *repository.OAuthRepository
}

func NewCampaignService(
	pool *pgxpool.Pool,
	campaigns *repository.CampaignRepository,
	outreach *repository.OutreachRepository,
	xlsx *XLSXService,
	templates *repository.TemplateRepository,
	resumes *repository.ResumeRepository,
	oauth *repository.OAuthRepository,
) *CampaignService {
	return &CampaignService{
		pool: pool, campaigns: campaigns, outreach: outreach, xlsx: xlsx,
		templates: templates, resumes: resumes, oauth: oauth,
	}
}

type CreateCampaignInput struct {
	XLSXFileID uuid.UUID
	TemplateID uuid.UUID
	StartAt    time.Time
}

func (s *CampaignService) Create(ctx context.Context, userID uuid.UUID, input CreateCampaignInput) (*model.Campaign, error) {
	if _, err := s.resumes.GetByUser(ctx, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrResumeRequired
		}
		return nil, err
	}

	hasOAuth, err := s.oauth.ExistsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !hasOAuth {
		return nil, ErrOAuthRequired
	}

	xlsxFile, err := s.xlsx.Get(ctx, input.XLSXFileID, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.templates.GetByID(ctx, input.TemplateID, userID); err != nil {
		return nil, err
	}

	parsed, err := s.xlsx.DownloadAndParse(ctx, xlsxFile)
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, fmt.Errorf("xlsx validation failed")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	campaignNum, err := s.campaigns.NextCampaignNumber(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	campaign := &model.Campaign{
		UserID:         userID,
		CampaignNumber: campaignNum,
		XLSXFileID:     input.XLSXFileID,
		TemplateID:     input.TemplateID,
		Status:         model.CampaignStatusScheduled,
		Total:          len(parsed.Rows),
		ScheduledAt:    input.StartAt,
	}
	if err := s.campaigns.Create(ctx, tx, campaign); err != nil {
		return nil, err
	}

	var outreachItems []model.Outreach
	for i, row := range parsed.Rows {
		outreachItems = append(outreachItems, model.Outreach{
			UserID:         userID,
			CampaignID:     campaign.ID,
			CompanyName:    row.CompanyName,
			Role:           row.Role,
			RecipientEmail: row.CompanyMail,
			Status:         model.OutreachStatusPending,
			ScheduledAt:    input.StartAt.Add(time.Duration(i) * outreachInterval),
		})
	}
	if err := s.outreach.CreateBatch(ctx, tx, outreachItems); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return campaign, nil
}

func (s *CampaignService) List(ctx context.Context, userID uuid.UUID) ([]model.Campaign, error) {
	return s.campaigns.ListByUser(ctx, userID)
}

func (s *CampaignService) Get(ctx context.Context, userID, id uuid.UUID) (*model.Campaign, error) {
	return s.campaigns.GetByID(ctx, id, userID)
}

func (s *CampaignService) Cancel(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.campaigns.Cancel(ctx, id, userID); err != nil {
		return err
	}
	return s.outreach.CancelPendingForCampaign(ctx, id)
}
