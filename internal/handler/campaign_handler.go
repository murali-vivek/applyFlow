package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/applyflow/applyflow/internal/middleware"
	apperrors "github.com/applyflow/applyflow/internal/errors"
	"github.com/applyflow/applyflow/internal/repository"
	"github.com/applyflow/applyflow/internal/service"
	"github.com/google/uuid"
)

type CampaignHandler struct {
	svc *service.CampaignService
}

func NewCampaignHandler(svc *service.CampaignService) *CampaignHandler {
	return &CampaignHandler{svc: svc}
}

type createCampaignRequest struct {
	XLSXFileID string `json:"xlsxFileId"`
	TemplateID string `json:"templateId"`
	StartAt    string `json:"startAt"`
}

func (h *CampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	var req createCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}

	xlsxID, err := uuid.Parse(req.XLSXFileID)
	if err != nil {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid xlsxFileId")
		return
	}
	templateID, err := uuid.Parse(req.TemplateID)
	if err != nil {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid templateId")
		return
	}

	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid startAt timestamp")
		return
	}

	campaign, err := h.svc.Create(r.Context(), userID, service.CreateCampaignInput{
		XLSXFileID: xlsxID,
		TemplateID: templateID,
		StartAt:    startAt,
	})
	if errors.Is(err, service.ErrResumeRequired) {
		apperrors.Write(w, http.StatusBadRequest, "RESUME_REQUIRED", err.Error())
		return
	}
	if errors.Is(err, service.ErrOAuthRequired) {
		apperrors.Write(w, http.StatusBadRequest, "OAUTH_REQUIRED", err.Error())
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		apperrors.Write(w, http.StatusBadRequest, "NOT_FOUND", "xlsx or template not found")
		return
	}
	if err != nil {
		apperrors.Write(w, http.StatusInternalServerError, "INTERNAL", "failed to create campaign")
		return
	}
	writeJSON(w, http.StatusCreated, campaign)
}

func (h *CampaignHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	campaigns, err := h.svc.List(r.Context(), userID)
	if err != nil {
		apperrors.Write(w, http.StatusInternalServerError, "INTERNAL", "failed to list campaigns")
		return
	}
	writeJSON(w, http.StatusOK, campaigns)
}

func (h *CampaignHandler) Get(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	userID := middleware.UserID(r.Context())
	campaign, err := h.svc.Get(r.Context(), userID, id)
	if errors.Is(err, repository.ErrNotFound) {
		apperrors.Write(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
		return
	}
	if err != nil {
		apperrors.Write(w, http.StatusInternalServerError, "INTERNAL", "failed to get campaign")
		return
	}
	writeJSON(w, http.StatusOK, campaign)
}

func (h *CampaignHandler) Cancel(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	userID := middleware.UserID(r.Context())
	if err := h.svc.Cancel(r.Context(), userID, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			apperrors.Write(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
			return
		}
		apperrors.Write(w, http.StatusInternalServerError, "INTERNAL", "failed to cancel campaign")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
