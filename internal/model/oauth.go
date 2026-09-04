package model

import (
	"time"

	"github.com/google/uuid"
)

type OAuthCredential struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"userId"`
	Provider     string    `json:"provider"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	CreatedAt    time.Time `json:"createdAt"`
	ModifiedAt   time.Time `json:"modifiedAt"`
}
