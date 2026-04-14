package http

import (
	"context"
	"net/http"
	"time"

	"github.com/nglong14/PromptBank/internal/asset"
	"github.com/nglong14/PromptBank/internal/llm"
)

// llmUnavailable writes a 503 when no LLM client is configured.
func llmUnavailable(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error": "LLM features are not configured (GEMINI_API_KEY not set)",
	})
}

// llmNormalizeHandler retouches fuzzy wizard answers into polished asset fields.
//
// POST /api/v1/llm/normalize
// Body: { "answers": { "goal": "...", "persona": "...", ... } }
func llmNormalizeHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.LLMClient == nil {
			llmUnavailable(w)
			return
		}

		var req struct {
			Answers map[string]string `json:"answers"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if len(req.Answers) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "answers must not be empty"})
			return
		}

		result, err := deps.LLMClient.Normalize(r.Context(), req.Answers)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

// llmSuggestFrameworkHandler picks the best framework for the given assets.
//
// POST /api/v1/llm/suggest-framework
// Body: { "assets": { ... } }
func llmSuggestFrameworkHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.LLMClient == nil {
			llmUnavailable(w)
			return
		}

		var req struct {
			Assets asset.Assets `json:"assets"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		suggestion, err := deps.LLMClient.SuggestFramework(r.Context(), req.Assets)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, suggestion)
	}
}

// llmScoreHandler silently scores a composed prompt on 4 quality dimensions.
//
// POST /api/v1/llm/score
// Body: { "composedOutput": "..." }
func llmScoreHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.LLMClient == nil {
			llmUnavailable(w)
			return
		}

		var req struct {
			ComposedOutput string `json:"composedOutput"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.ComposedOutput == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "composedOutput must not be empty"})
			return
		}

		score, err := deps.LLMClient.Score(r.Context(), req.ComposedOutput)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, score)
	}
}

// llmRefineHandler runs the iterative refinement agent. The refiner can hold
// a connection open for up to 90 seconds (8 tool-call iterations × ~4 s each),
// so a per-handler timeout is applied instead of relying on a global WriteTimeout.
//
// POST /api/v1/llm/refine
// Body: { "assets": {...}, "composedOutput": "...", "userFeedback": "...", "history": [...] }
func llmRefineHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.LLMClient == nil {
			llmUnavailable(w)
			return
		}

		// 90 s cap: bounds the refiner's max wall time. A global WriteTimeout is
		// intentionally not set on the server because it would break this endpoint.
		ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
		defer cancel()

		var req llm.RefineRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.UserFeedback == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userFeedback must not be empty"})
			return
		}

		result, err := deps.LLMClient.Refine(ctx, req)
		if err != nil {
			if ctx.Err() != nil {
				writeJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "refinement timed out"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}
