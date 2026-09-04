package model

import (
	"time"

	"github.com/google/uuid"
)

type Template struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	Name      string    `json:"name"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Variables []string  `json:"variables"`
	IsDefault bool      `json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}
