// Retouch fuzzy phrases into polished, complete, and consistent asset fields
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nglong14/PromptBank/internal/asset"
	"github.com/nglong14/PromptBank/internal/framework"
)

type NormalizeResponse struct {
	Assets               asset.Assets `json:"assets"`
	Confidence           float64      `json:"confidence"`
	SuggestedFrameworkID string       `json:"suggestedFrameworkId,omitempty"`
}

const normalizerSystemPrompt = `You are a professional prompt engineer assistant helping non-technical users build AI prompts.

The user has answered a series of plain-English questions about the prompt they want to create.
Their answers may be fuzzy, incomplete, or conversational — your job is to retouch them into polished, professional asset field values.

Rules:
1. Expand incomplete thoughts into full, clear sentences.
2. Fix grammar and clarify vague language.
3. Keep the user's intent — never invent facts they didn't imply.
4. If the user left a field blank or it is clearly missing, return an empty string for that field.
5. "examples" is an array; if the user provided example input/output text, parse it into the array format.

Return ONLY a valid JSON object with these exact keys:
{
  "persona": "...",
  "context": "...",
  "tone": "...",
  "constraints": "...",
  "goal": "...",
  "examples": [{"input": "...", "output": "..."}]
}

Do not include any explanation or markdown fences — just the JSON object.`

// Retouch the user's raw wizard answers into clean asset fields, 
// return the polished assets, a confidence score, and (if >= 0.6) the best-matching framework ID determined by the rule engine
func (c *Client) Normalize(ctx context.Context, answers map[string]string) (NormalizeResponse, error) {
	userPrompt := buildAnswersPrompt(answers)

	text, err := c.Generate(ctx, normalizerSystemPrompt, userPrompt)
	if err != nil {
		return NormalizeResponse{}, fmt.Errorf("normalize: %w", err)
	}

	text = cleanJSON(text)
	var assets asset.Assets
	if err := json.Unmarshal([]byte(text), &assets); err != nil {
		return NormalizeResponse{}, fmt.Errorf("parse normalize response: %w (raw: %.200s)", err, text)
	}

	confidence := computeNormalizeConfidence(assets)

	var suggestedID string
	if confidence >= 0.6 {
		bestID, _ := ruleEngineScore(assets)
		suggestedID = bestID
	}

	return NormalizeResponse{
		Assets:               assets,
		Confidence:           confidence,
		SuggestedFrameworkID: suggestedID,
	}, nil
}

// Formats the wizard answers into a readable prompt for Gemini.
func buildAnswersPrompt(answers map[string]string) string {
	order := []struct{ key, label string }{
		{"goal", "What the prompt should achieve"},
		{"persona", "Who the AI should be"},
		{"context", "Background the AI should know"},
		{"tone", "Tone or style"},
		{"constraints", "Rules to follow or avoid"},
		{"examples", "Example input and output"},
	}

	var sb strings.Builder	// Build string
	sb.WriteString("Here are the user's answers to the prompt-building questions:\n\n")
	for _, field := range order {
		val := strings.TrimSpace(answers[field.key])
		if val == "" {
			val = "(not provided)"
		}
		fmt.Fprintf(&sb, "**%s:**\n%s\n\n", field.label, val)
	}
	sb.WriteString("Please retouch these answers into polished asset field values and return the JSON object.")
	return sb.String()
}

// Returns the fraction of non-empty text fields.
func computeNormalizeConfidence(a asset.Assets) float64 {
	total := 5 // persona, context, tone, constraints, goal
	filled := 0
	if strings.TrimSpace(a.Persona) != "" {
		filled++
	}
	if strings.TrimSpace(a.Context) != "" {
		filled++
	}
	if strings.TrimSpace(a.Tone) != "" {
		filled++
	}
	if strings.TrimSpace(a.Constraints) != "" {
		filled++
	}
	if strings.TrimSpace(a.Goal) != "" {
		filled++
	}
	return float64(filled) / float64(total)
}

// Computes the best-matching framework for the given assets
func ruleEngineScore(assets asset.Assets) (bestID string, bestScore float64) {
	normalized := asset.Normalize(assets)
	report := normalized.FieldReport

	isComplete := func(field string) bool {
		return report[field] == asset.QualityComplete
	}

	for _, fw := range framework.List() {
		required := 0
		completed := 0
		for _, slot := range fw.Slots {
			if !slot.Required {
				continue
			}
			required++
			if isComplete(slot.AssetField) {
				completed++
			}
		}
		if required == 0 {
			continue
		}
		score := float64(completed) / float64(required)
		if score > bestScore {
			bestScore = score
			bestID = fw.ID
		}
	}
	return bestID, bestScore
}
