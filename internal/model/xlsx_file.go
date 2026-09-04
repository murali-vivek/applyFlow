package model

import (
	"time"

	"github.com/google/uuid"
)

type XLSXFile struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"userId"`
	FileName    string    `json:"fileName"`
	StoragePath string    `json:"storagePath"`
	UploadedAt  time.Time `json:"uploadedAt"`
}

type XLSXRow struct {
	CompanyName string `json:"companyName"`
	Role        string `json:"role"`
	CompanyMail string `json:"companyMail"`
}
