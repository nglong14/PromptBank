package http

import (
	"net/http"
	"strings"

	"github.com/nglong14/PromptBank/internal/security"
)

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func registerHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		email := strings.TrimSpace(strings.ToLower(req.Email))
		if email == "" || len(req.Password) < 8 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password(min 8 chars) are required"})
			return
		}

		hash, err := security.HashPassword(req.Password)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
			return
		}

		user, err := deps.UserRepo.Create(r.Context(), email, hash)
		if err != nil {
			writeJSON(w, errorStatus(err), map[string]string{"error": "failed to create user"})
			return
		}

		token, err := deps.JWTManager.Issue(user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to issue token"})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"user":  user,
			"token": token,
		})
	}
}

func loginHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		email := strings.TrimSpace(strings.ToLower(req.Email))
		if email == "" || req.Password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
			return
		}

		user, err := deps.UserRepo.FindByEmail(r.Context(), email)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}

		if err := security.CheckPassword(user.PasswordHash, req.Password); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}

		token, err := deps.JWTManager.Issue(user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to issue token"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"user":  user,
			"token": token,
		})
	}
}
