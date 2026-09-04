package handler

import (
	"errors"
	"net/http"

	"github.com/applyflow/applyflow/internal/middleware"
	apperrors "github.com/applyflow/applyflow/internal/errors"
	"github.com/applyflow/applyflow/internal/repository"
	"github.com/applyflow/applyflow/internal/service"
)

type ResumeHandler struct {
	svc *service.ResumeService
}

func NewResumeHandler(svc *service.ResumeService) *ResumeHandler {
	return &ResumeHandler{svc: svc}
}

func (h *ResumeHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	resume, err := h.svc.Get(r.Context(), userID)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	if err != nil {
		apperrors.Write(w, http.StatusInternalServerError, "INTERNAL", "failed to get resume")
		return
	}
	writeJSON(w, http.StatusOK, resume)
}

func (h *ResumeHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_REQUEST", "file is required")
		return
	}
	defer file.Close()

	resume, err := h.svc.Upload(r.Context(), userID, header.Filename, file)
	if err != nil {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_RESUME", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resume)
}
