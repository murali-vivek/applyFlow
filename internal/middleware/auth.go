package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/applyflow/applyflow/internal/auth"
	apperrors "github.com/applyflow/applyflow/internal/errors"
	"github.com/google/uuid"
)

type contextKey string

const UserIDKey contextKey = "userID"
const UserEmailKey contextKey = "userEmail"

func Auth(jwt *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				apperrors.Write(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}
			claims, err := jwt.Parse(token)
			if err != nil {
				apperrors.Write(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(ctx context.Context) uuid.UUID {
	if v, ok := ctx.Value(UserIDKey).(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}

func extractToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if strings.HasPrefix(h, "Bearer ") {
			return strings.TrimPrefix(h, "Bearer ")
		}
	}
	if c, err := r.Cookie("token"); err == nil {
		return c.Value
	}
	return ""
}

func CORS(frontendURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", frontendURL)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
