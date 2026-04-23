package http

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nglong14/PromptBank/internal/diff"
)

// GET /api/v1/prompts/{promptID}/versions/diff?from=X&to=Y
func getVersionDiffHandler(deps Dependencies) http.HandlerFunc {
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

		fromNum, toNum, err := parseFromTo(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if fromNum == toNum {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "from and to must differ"})
			return
		}

		fromVer, err := deps.PromptRepo.GetVersionByNumber(r.Context(), promptID, fromNum)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("version %d not found", fromNum)})
			return
		}
		toVer, err := deps.PromptRepo.GetVersionByNumber(r.Context(), promptID, toNum)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("version %d not found", toNum)})
			return
		}

		result, err := diff.Compute(fromVer, toVer)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "diff computation failed"})
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

// parseFromTo reads the 'from' and 'to' query parameters and validates them.
func parseFromTo(r *http.Request) (int, int, error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		return 0, 0, fmt.Errorf("query params 'from' and 'to' are required")
	}
	from, err := strconv.Atoi(fromStr)
	if err != nil || from < 1 {
		return 0, 0, fmt.Errorf("'from' must be a positive integer")
	}
	to, err := strconv.Atoi(toStr)
	if err != nil || to < 1 {
		return 0, 0, fmt.Errorf("'to' must be a positive integer")
	}
	return from, to, nil
}
