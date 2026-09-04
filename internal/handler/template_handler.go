package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/applyflow/applyflow/internal/middleware"
	apperrors "github.com/applyflow/applyflow/internal/errors"
	"github.com/applyflow/applyflow/internal/service"
	"github.com/google/uuid"
)

type TemplateHandler struct {
	svc *service.TemplateService
}

func NewTemplateHandler(svc *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{svc: svc}
}

func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	templates, err := h.svc.List(r.Context(), userID)
	if err != nil {
		apperrors.Write(w, http.StatusInternalServerError, "INTERNAL", "failed to list templates")
		return
	}
	writeJSON(w, http.StatusOK, templates)
}

type templateRequest struct {
	Name      string `json:"name"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	IsDefault bool   `json:"isDefault"`
}

func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	var req templateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	t, err := h.svc.Create(r.Context(), userID, req.Name, req.Subject, req.Body, req.IsDefault)
	if errors.Is(err, service.ErrMaxTemplates) {
		apperrors.Write(w, http.StatusBadRequest, "MAX_TEMPLATES", err.Error())
		return
	}
	if errors.Is(err, service.ErrMissingCompanyVar) {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_TEMPLATE", err.Error())
		return
	}
	if err != nil {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_TEMPLATE", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	userID := middleware.UserID(r.Context())
	var req templateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	t, err := h.svc.Update(r.Context(), userID, id, req.Name, req.Subject, req.Body, req.IsDefault)
	if err != nil {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_TEMPLATE", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	userID := middleware.UserID(r.Context())
	if err := h.svc.Delete(r.Context(), userID, id); err != nil {
		if errors.Is(err, service.ErrCannotDeleteDefault) {
			apperrors.Write(w, http.StatusBadRequest, "CANNOT_DELETE", err.Error())
			return
		}
		apperrors.Write(w, http.StatusNotFound, "NOT_FOUND", "template not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
