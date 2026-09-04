package model

import (
	"time"

	"github.com/google/uuid"
)

type Resume struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"userId"`
	Name       string    `json:"name"`
	S3Path     string    `json:"s3Path"`
	UploadedAt time.Time `json:"uploadedAt"`
}
