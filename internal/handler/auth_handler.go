package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"

	"github.com/applyflow/applyflow/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	state := randomState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   300,
	})
	http.Redirect(w, r, h.auth.AuthURL(state), http.StatusTemporaryRedirect)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	token, err := h.auth.HandleCallback(r.Context(), code)
	if err != nil {
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	redirectURL := h.auth.FrontendURL() + "/auth/callback?token=" + url.QueryEscape(token)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "authenticated"})
}

func randomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	redirect := h.auth.FrontendURL()
	u, _ := url.Parse(redirect)
	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}
