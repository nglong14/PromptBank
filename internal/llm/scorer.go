// Score the composed prompt on four dimensions: clarity, completeness, specificity, and actionability
package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

// QualityScore is the result of silently scoring a composed prompt.
type QualityScore struct {
	Score    float64 `json:"score"`
	Feedback string  `json:"feedback"`
}

const scorerSystemPrompt = `You are an expert prompt quality reviewer. Evaluate the given composed prompt
on four dimensions, each scored from 1 to 10:

- Clarity: Is the prompt unambiguous? Would different people read it the same way?
- Completeness: Does it include enough context for the AI to respond well without guessing?
- Specificity: Are the instructions precise, or too vague/generic?
- Actionability: Does it give the AI a clear, concrete task to execute?

Compute the final score as the average of the four dimensions, rounded to one decimal place.
Write feedback as 1–2 sentences: one strength, one improvement suggestion.

Return ONLY a valid JSON object:
{
  "score": <number 1.0–10.0>,
  "feedback": "<1-2 sentences>"
}

Do not include any explanation or markdown fences — just the JSON object.`

// Score silently evaluates a composed prompt and returns a quality score with feedback.
// It is designed to be called in a goroutine after compose; errors are non-fatal.
func (c *Client) Score(ctx context.Context, composedOutput string) (QualityScore, error) {
	userPrompt := fmt.Sprintf("Please evaluate the following composed prompt:\n\n---\n%s\n---", composedOutput)

	text, err := c.Generate(ctx, scorerSystemPrompt, userPrompt)
	if err != nil {
		return QualityScore{}, fmt.Errorf("score: %w", err)
	}

	text = cleanJSON(text)
	var score QualityScore
	if err := json.Unmarshal([]byte(text), &score); err != nil {
		return QualityScore{}, fmt.Errorf("parse score response: %w (raw: %.200s)", err, text)
	}
	return score, nil
}
