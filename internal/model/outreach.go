package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	OutreachStatusPending    = "PENDING"
	OutreachStatusQueued     = "QUEUED"
	OutreachStatusProcessing = "PROCESSING"
	OutreachStatusSent       = "SENT"
	OutreachStatusFailed     = "FAILED"
	OutreachStatusCancelled  = "CANCELLED"
)

type Outreach struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"userId"`
	CampaignID      uuid.UUID  `json:"campaignId"`
	CompanyName     string     `json:"companyName"`
	Role            string     `json:"role"`
	RecipientEmail  string     `json:"recipientEmail"`
	Status          string     `json:"status"`
	ScheduledAt     time.Time  `json:"scheduledAt"`
	SentAt          *time.Time `json:"sentAt"`
	FailedAt        *time.Time `json:"failedAt"`
	GmailMessageID  *string    `json:"gmailMessageId"`
	ErrorMessage    *string    `json:"errorMessage"`
	CreatedAt       time.Time  `json:"createdAt"`
}
