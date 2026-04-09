package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/nglong14/PromptBank/internal/security"
)

type contextKey string

const userIDContextKey contextKey = "userID"

func authMiddleware(jwtManager *security.JWTManager, tokenPrefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
				return
			}

			prefix := tokenPrefix + " "
			if !strings.HasPrefix(header, prefix) {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authorization scheme"})
				return
			}

			token := strings.TrimPrefix(header, prefix)
			claims, err := jwtManager.Parse(token)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}

			// parse user id from claims
			userID, err := uuid.Parse(claims.UserID)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user identity"})
				return
			}

			// set user id in context
			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func userIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(userIDContextKey)
	userID, ok := v.(uuid.UUID)
	return userID, ok
}
