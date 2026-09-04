package handler

import (
	"net/http"

	"github.com/applyflow/applyflow/internal/middleware"
	apperrors "github.com/applyflow/applyflow/internal/errors"
	"github.com/applyflow/applyflow/internal/service"
)

type XLSXHandler struct {
	svc *service.XLSXService
}

func NewXLSXHandler(svc *service.XLSXService) *XLSXHandler {
	return &XLSXHandler{svc: svc}
}

func (h *XLSXHandler) Upload(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.svc.Upload(r.Context(), userID, header.Filename, file)
	if err != nil {
		apperrors.Write(w, http.StatusBadRequest, "INVALID_XLSX", err.Error())
		return
	}
	if !result.Valid {
		writeJSON(w, http.StatusOK, result)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
