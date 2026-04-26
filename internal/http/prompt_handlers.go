package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nglong14/PromptBank/internal/repository"
)

type createPromptRequest struct {
	Title    string   `json:"title"`
	Status   string   `json:"status"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

type updatePromptRequest struct {
	Title    string   `json:"title"`
	Status   string   `json:"status"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

type createPromptVersionRequest struct {
	Assets          json.RawMessage `json:"assets"`
	FrameworkID     string          `json:"frameworkId"`
	TechniqueIDs    []string        `json:"techniqueIds"`
	ComposedOutput  string          `json:"composedOutput"`
	ChangeType      string          `json:"changeType,omitempty"`
	ChangeSummary   string          `json:"changeSummary,omitempty"`
	ParentVersionID string          `json:"parentVersionId,omitempty"`
}

type derivePromptRequest struct {
	SourcePromptID  string `json:"sourcePromptId"`
	SourceVersionID string `json:"sourceVersionId,omitempty"`
	NewTitle        string `json:"newTitle"`
}

func listPromptsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing user identity"})
			return
		}

		prompts, err := deps.PromptRepo.ListByOwner(r.Context(), userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list prompts"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"items": prompts})
	}
}

func createPromptHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing user identity"})
			return
		}

		var req createPromptRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if strings.TrimSpace(req.Title) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
			return
		}
		if req.Status == "" {
			req.Status = "draft"
		}

		prompt, err := deps.PromptRepo.Create(r.Context(), repository.CreatePromptInput{
			Title:    strings.TrimSpace(req.Title),
			Status:   req.Status,
			Category: strings.TrimSpace(req.Category),
			Tags:     req.Tags,
			OwnerID:  userID,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create prompt"})
			return
		}

		writeJSON(w, http.StatusCreated, prompt)
	}
}

func getPromptHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing user identity"})
			return
		}

		promptID, err := uuid.Parse(chi.URLParam(r, "promptID"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid prompt id"})
			return
		}

		prompt, err := deps.PromptRepo.GetByID(r.Context(), promptID, userID)
		if err != nil {
			writeJSON(w, errorStatus(err), map[string]string{"error": "prompt not found"})
			return
		}

		writeJSON(w, http.StatusOK, prompt)
	}
}

func updatePromptHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing user identity"})
			return
		}

		promptID, err := uuid.Parse(chi.URLParam(r, "promptID"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid prompt id"})
			return
		}

		var req updatePromptRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if strings.TrimSpace(req.Title) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
			return
		}
		if req.Status == "" {
			req.Status = "draft"
		}

		prompt, err := deps.PromptRepo.Update(
			r.Context(),
			promptID,
			userID,
			strings.TrimSpace(req.Title),
			req.Status,
			strings.TrimSpace(req.Category),
			req.Tags,
		)
		if err != nil {
			writeJSON(w, errorStatus(err), map[string]string{"error": "failed to update prompt"})
			return
		}

		writeJSON(w, http.StatusOK, prompt)
	}
}

func createPromptVersionHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing user identity"})
			return
		}

		promptID, err := uuid.Parse(chi.URLParam(r, "promptID"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid prompt id"})
			return
		}

		if _, err := deps.PromptRepo.GetByID(r.Context(), promptID, userID); err != nil {
			writeJSON(w, errorStatus(err), map[string]string{"error": "prompt not found"})
			return
		}

		var req createPromptVersionRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if len(req.Assets) == 0 {
			req.Assets = json.RawMessage(`{}`)
		}

		// 'fork' attribution belongs to DerivePrompt; reject it here so we don't end up
		// with fork-typed rows that lack a real cross-prompt source.
		if req.ChangeType == "fork" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "use the derive endpoint to create a fork"})
			return
		}

		var parentVersionID *uuid.UUID
		if strings.TrimSpace(req.ParentVersionID) != "" {
			parsed, err := uuid.Parse(req.ParentVersionID)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid parentVersionId"})
				return
			}
			parentVersionID = &parsed
		}

		var changeSummary *string
		if s := strings.TrimSpace(req.ChangeSummary); s != "" {
			changeSummary = &s
		}

		version, err := deps.PromptRepo.CreateVersion(r.Context(), repository.CreateVersionInput{
			PromptID:        promptID,
			Assets:          req.Assets,
			FrameworkID:     req.FrameworkID,
			TechniqueIDs:    req.TechniqueIDs,
			ComposedOut:     req.ComposedOutput,
			CreatedBy:       userID,
			ChangeType:      req.ChangeType,
			ChangeSummary:   changeSummary,
			ParentVersionID: parentVersionID,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create version"})
			return
		}
		writeJSON(w, http.StatusCreated, version)
	}
}

func listPromptVersionsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing user identity"})
			return
		}

		promptID, err := uuid.Parse(chi.URLParam(r, "promptID"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid prompt id"})
			return
		}

		versions, err := deps.PromptRepo.ListVersions(r.Context(), promptID, userID)
		if err != nil {
			writeJSON(w, errorStatus(err), map[string]string{"error": "failed to list versions"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"items": versions})
	}
}

func derivePromptHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing user identity"})
			return
		}

		var req derivePromptRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		sourcePromptID, err := uuid.Parse(req.SourcePromptID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sourcePromptId"})
			return
		}
		if strings.TrimSpace(req.NewTitle) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "newTitle is required"})
			return
		}

		var sourceVersionID *uuid.UUID
		if req.SourceVersionID != "" {
			parsed, err := uuid.Parse(req.SourceVersionID)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sourceVersionId"})
				return
			}
			sourceVersionID = &parsed
		}

		prompt, version, err := deps.PromptRepo.DerivePrompt(r.Context(), repository.DerivePromptInput{
			SourcePromptID:  sourcePromptID,
			SourceVersionID: sourceVersionID,
			NewTitle:        strings.TrimSpace(req.NewTitle),
			OwnerID:         userID,
		})
		if err != nil {
			writeJSON(w, errorStatus(err), map[string]string{"error": "failed to derive prompt"})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"prompt":  prompt,
			"version": version,
		})
	}
}
