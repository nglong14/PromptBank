package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/nglong14/PromptBank/internal/repository"
	"github.com/nglong14/PromptBank/internal/security"
)

// Dependencies for the HTTP router
type Dependencies struct {
	UserRepo    *repository.UserRepository
	PromptRepo  *repository.PromptRepository
	JWTManager  *security.JWTManager
	TokenPrefix string
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Post("/api/v1/auth/register", registerHandler(deps))
	r.Post("/api/v1/auth/login", loginHandler(deps))

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware(deps.JWTManager, deps.TokenPrefix))

		r.Get("/prompts", listPromptsHandler(deps))
		r.Post("/prompts", createPromptHandler(deps))
		r.Get("/prompts/{promptID}", getPromptHandler(deps))
		r.Patch("/prompts/{promptID}", updatePromptHandler(deps))
		r.Post("/prompts/{promptID}/versions", createPromptVersionHandler(deps))
		r.Get("/prompts/{promptID}/versions", listPromptVersionsHandler(deps))
		r.Post("/prompts/derive", derivePromptHandler(deps))

		r.Get("/frameworks", listFrameworksHandler())
		r.Get("/techniques", listTechniquesHandler())
		r.Post("/assets/normalize", normalizeAssetsHandler())
		r.Post("/assets/validate", validateSlotsHandler())
		r.Post("/compose", composeHandler())
	})

	return r
}

// Write JSON response
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func errorStatus(err error) int {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return http.StatusNotFound
	case strings.Contains(strings.ToLower(err.Error()), "duplicate key"):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
