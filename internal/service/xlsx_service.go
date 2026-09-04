package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/applyflow/applyflow/internal/model"
	"github.com/applyflow/applyflow/internal/repository"
	"github.com/applyflow/applyflow/internal/storage"
	"github.com/applyflow/applyflow/internal/xlsx"
	"github.com/google/uuid"
)

const maxXLSXSize = 10 << 20

type XLSXService struct {
	xlsxRepo *repository.XLSXRepository
	s3       *storage.S3Client
}

func NewXLSXService(xlsxRepo *repository.XLSXRepository, s3 *storage.S3Client) *XLSXService {
	return &XLSXService{xlsxRepo: xlsxRepo, s3: s3}
}

type XLSXUploadResult struct {
	ID        uuid.UUID        `json:"id,omitempty"`
	FileName  string           `json:"fileName"`
	RowCount  int              `json:"rowCount"`
	TotalRows int              `json:"totalRows"`
	Skipped   xlsx.RowSkipStats `json:"skipped"`
	Valid     bool             `json:"valid"`
	Errors    []string         `json:"errors,omitempty"`
	Preview   []model.XLSXRow  `json:"preview,omitempty"`
}

func (s *XLSXService) Upload(ctx context.Context, userID uuid.UUID, fileName string, r io.Reader) (*XLSXUploadResult, error) {
	if !strings.HasSuffix(strings.ToLower(fileName), ".xlsx") {
		return nil, fmt.Errorf("only XLSX files are allowed")
	}

	data, err := io.ReadAll(io.LimitReader(r, maxXLSXSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxXLSXSize {
		return nil, fmt.Errorf("file exceeds 10 MB limit")
	}

	result, err := xlsx.ParseAndValidate(data)
	if err != nil {
		return nil, err
	}
	if !result.Valid {
		return &XLSXUploadResult{
			FileName:  fileName,
			RowCount:  result.RowCount,
			TotalRows: result.TotalRows,
			Skipped:   result.Skipped,
			Valid:     false,
			Errors:    result.Errors,
			Preview:   previewRows(result.Rows, 5),
		}, nil
	}

	key := fmt.Sprintf("users/%s/xlsx/%s.xlsx", userID, uuid.New())
	if err := s.s3.Upload(ctx, key, bytes.NewReader(data), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"); err != nil {
		return nil, fmt.Errorf("upload xlsx: %w", err)
	}

	f := &model.XLSXFile{
		UserID:      userID,
		FileName:    fileName,
		StoragePath: key,
	}
	if err := s.xlsxRepo.Create(ctx, f); err != nil {
		return nil, err
	}

	return &XLSXUploadResult{
		ID:        f.ID,
		FileName:  fileName,
		RowCount:  result.RowCount,
		TotalRows: result.TotalRows,
		Skipped:   result.Skipped,
		Valid:     true,
		Preview:   previewRows(result.Rows, 5),
	}, nil
}

func previewRows(rows []model.XLSXRow, limit int) []model.XLSXRow {
	if len(rows) == 0 {
		return []model.XLSXRow{}
	}
	if len(rows) <= limit {
		return rows
	}
	return rows[:limit]
}

func (s *XLSXService) Get(ctx context.Context, id, userID uuid.UUID) (*model.XLSXFile, error) {
	return s.xlsxRepo.GetByID(ctx, id, userID)
}

func (s *XLSXService) DownloadAndParse(ctx context.Context, f *model.XLSXFile) (*xlsx.ValidationResult, error) {
	data, err := s.s3.Download(ctx, f.StoragePath)
	if err != nil {
		return nil, err
	}
	return xlsx.ParseAndValidate(data)
}
