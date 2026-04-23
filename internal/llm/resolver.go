// Suggest the best framework for the given assets
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nglong14/PromptBank/internal/asset"
	"github.com/nglong14/PromptBank/internal/framework"
)

type FrameworkSuggestion struct {
	FrameworkID string  `json:"frameworkId"`
	Confidence  float64 `json:"confidence"`
	Rationale   string  `json:"rationale"`
}

const resolverSystemPrompt = `You are a prompt engineering expert. You will be given a set of prompt asset fields
and a list of prompt frameworks. Your job is to choose the best-fitting framework.

Return ONLY a valid JSON object with these exact keys:
{
  "frameworkId": "<id of the chosen framework>",
  "confidence": <a number between 0 and 1 representing your confidence>,
  "rationale": "<one sentence explaining why this framework fits best>"
}

Do not include any explanation or markdown fences — just the JSON object.`

// Pick the best framework for the given assets, return the framework ID, confidence, and rationale
func (c *Client) SuggestFramework(ctx context.Context, assets asset.Assets) (FrameworkSuggestion, error) {
	bestID, score := ruleEngineScore(assets)

	if score >= 0.6 {
		return FrameworkSuggestion{
			FrameworkID: bestID,
			Confidence:  score,
			Rationale:   "Selected automatically based on which framework's required fields are best covered by your prompt assets.",
		}, nil
	}

	// Low confidence — call Gemini to resolve ambiguity
	userPrompt := buildResolverPrompt(assets)
	text, err := c.Generate(ctx, resolverSystemPrompt, userPrompt)
	if err != nil {
		return FrameworkSuggestion{}, fmt.Errorf("suggest framework: %w", err)
	}

	text = cleanJSON(text)
	var suggestion FrameworkSuggestion
	if err := json.Unmarshal([]byte(text), &suggestion); err != nil {
		return FrameworkSuggestion{}, fmt.Errorf("parse framework suggestion: %w (raw: %.200s)", err, text)
	}
	return suggestion, nil
}

// Constructs the user message for the LLM resolver.
func buildResolverPrompt(assets asset.Assets) string {
	var sb strings.Builder
	sb.WriteString("Here are the current prompt asset fields:\n\n")
	writeField(&sb, "Goal", assets.Goal)
	writeField(&sb, "Persona", assets.Persona)
	writeField(&sb, "Context", assets.Context)
	writeField(&sb, "Tone", assets.Tone)
	writeField(&sb, "Constraints", assets.Constraints)
	if len(assets.Examples) > 0 {
		fmt.Fprintf(&sb, "**Examples:** %d example(s) provided\n", len(assets.Examples))
	}

	sb.WriteString("\nAvailable frameworks:\n\n")
	for _, fw := range framework.List() {
		fmt.Fprintf(&sb, "**%s** (id: %s): %s\n", fw.Name, fw.ID, fw.Description)
		fmt.Fprintf(&sb, "  Required slots: ")
		var required []string
		for _, slot := range fw.Slots {
			if slot.Required {
				required = append(required, fmt.Sprintf("%s (%s)", slot.Name, slot.Description))
			}
		}
		sb.WriteString(strings.Join(required, ", "))
		sb.WriteString("\n\n")
	}

	sb.WriteString("Which framework best fits these assets? Return the JSON object.")
	return sb.String()
}

func writeField(sb *strings.Builder, label, value string) {
	if strings.TrimSpace(value) == "" {
		fmt.Fprintf(sb, "**%s:** (not provided)\n", label)
	} else {
		fmt.Fprintf(sb, "**%s:** %s\n", label, value)
	}
}
