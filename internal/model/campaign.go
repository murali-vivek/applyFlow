package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	CampaignStatusDraft     = "DRAFT"
	CampaignStatusScheduled = "SCHEDULED"
	CampaignStatusRunning   = "RUNNING"
	CampaignStatusCompleted = "COMPLETED"
	CampaignStatusCancelled = "CANCELLED"
)

type Campaign struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"userId"`
	CampaignNumber int        `json:"campaignNumber"`
	XLSXFileID     uuid.UUID  `json:"xlsxFileId"`
	TemplateID     uuid.UUID  `json:"templateId"`
	Status         string     `json:"status"`
	Total          int        `json:"total"`
	Sent           int        `json:"sent"`
	Failed         int        `json:"failed"`
	CreatedAt      time.Time  `json:"createdAt"`
	ScheduledAt    time.Time  `json:"scheduledAt"`
	StartedAt      *time.Time `json:"startedAt"`
	FinishedAt     *time.Time `json:"finishedAt"`
}
