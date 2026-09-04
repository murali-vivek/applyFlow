package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/applyflow/applyflow/internal/model"
	"github.com/applyflow/applyflow/internal/repository"
	"github.com/applyflow/applyflow/internal/storage"
	"github.com/google/uuid"
)

const maxResumeSize = 10 << 20 // 10 MB

type ResumeService struct {
	resumes *repository.ResumeRepository
	s3      *storage.S3Client
}

func NewResumeService(resumes *repository.ResumeRepository, s3 *storage.S3Client) *ResumeService {
	return &ResumeService{resumes: resumes, s3: s3}
}

func (s *ResumeService) Get(ctx context.Context, userID uuid.UUID) (*model.Resume, error) {
	return s.resumes.GetByUser(ctx, userID)
}

func (s *ResumeService) Upload(ctx context.Context, userID uuid.UUID, fileName string, r io.Reader) (*model.Resume, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext != ".pdf" {
		return nil, fmt.Errorf("only PDF files are allowed")
	}

	data, err := io.ReadAll(io.LimitReader(r, maxResumeSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file is empty")
	}
	if len(data) > maxResumeSize {
		return nil, fmt.Errorf("file exceeds 10 MB limit")
	}
	if !isPDF(data) {
		return nil, fmt.Errorf("invalid PDF file")
	}

	key := fmt.Sprintf("users/%s/resume/%s.pdf", userID, uuid.New())
	if err := s.s3.Upload(ctx, key, bytes.NewReader(data), "application/pdf"); err != nil {
		return nil, fmt.Errorf("upload resume: %w", err)
	}

	resume := &model.Resume{
		UserID: userID,
		Name:   fileName,
		S3Path: key,
	}
	if err := s.resumes.Upsert(ctx, resume); err != nil {
		return nil, err
	}
	return resume, nil
}

func isPDF(data []byte) bool {
	return len(data) >= 4 && string(data[:4]) == "%PDF"
}
