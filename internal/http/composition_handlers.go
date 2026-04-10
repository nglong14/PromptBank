package http

import (
	"net/http"

	"github.com/nglong14/PromptBank/internal/asset"
	"github.com/nglong14/PromptBank/internal/composition"
	"github.com/nglong14/PromptBank/internal/framework"
	"github.com/nglong14/PromptBank/internal/technique"
	"github.com/nglong14/PromptBank/internal/validation"
)

func listFrameworksHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": framework.List()})
	}
}

func listTechniquesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": technique.List()})
	}
}

type normalizeAssetsRequest struct {
	Assets asset.Assets `json:"assets"`
}

func normalizeAssetsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req normalizeAssetsRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		normalized := asset.Normalize(req.Assets)
		writeJSON(w, http.StatusOK, normalized)
	}
}

type validateSlotsRequest struct {
	Assets      asset.Assets `json:"assets"`
	FrameworkID string       `json:"frameworkId"`
}

func validateSlotsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req validateSlotsRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		fw, ok := framework.Get(req.FrameworkID)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown framework"})
			return
		}

		normalized := asset.Normalize(req.Assets)
		diags := validation.ValidateSlots(fw, normalized)

		writeJSON(w, http.StatusOK, map[string]any{
			"diagnostics": diags,
			"hasErrors":   validation.HasErrors(diags),
		})
	}
}

func composeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := userIDFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing user identity"})
			return
		}

		var req composition.ComposeInput
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		result, err := composition.Compose(req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}
