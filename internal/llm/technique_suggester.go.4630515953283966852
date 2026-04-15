package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nglong14/PromptBank/internal/asset"
)

type SuggestTechniqueRequest struct {
	TechniqueID string       `json:"techniqueId"`
	Assets      asset.Assets `json:"assets"`
}

type SuggestTechniqueResponse struct {
	Assets asset.Assets `json:"assets"`
}

const fewShotSystemPrompt = `You are a prompt engineering assistant.
Given the user's goal and context, generate 2-3 high-quality example input/output pairs that demonstrate the desired behavior.

Return ONLY a valid JSON array of objects with "input" and "output" keys:
[{"input": "...", "output": "..."}, ...]

Do not include any explanation or markdown fences — just the JSON array.`

const roleSystemPrompt = `You are a prompt engineering assistant.
Given the user's goal and context, write a detailed, professional persona description for the AI to adopt.
The persona should specify expertise, communication style, and relevant background.

Return ONLY the persona text as a plain string (no JSON, no markdown fences, no quotes around it).`

const constraintsSystemPrompt = `You are a prompt engineering assistant.
Given the user's goal and context, extract 3-5 clear, actionable constraints that should bound the AI's output.
Each constraint should be on its own line, starting with a dash.

Return ONLY the constraints text as a plain string (no JSON, no markdown fences).
Example format:
- Do not exceed 200 words
- Use formal academic tone
- Cite sources when making factual claims`

func (c *Client) SuggestTechniqueAssets(ctx context.Context, req SuggestTechniqueRequest) (SuggestTechniqueResponse, error) {
	merged := req.Assets

	userPrompt := buildTechniqueUserPrompt(req.Assets)

	switch req.TechniqueID {
	case "few-shot":
		text, err := c.Generate(ctx, fewShotSystemPrompt, userPrompt)
		if err != nil {
			return SuggestTechniqueResponse{}, fmt.Errorf("suggest few-shot: %w", err)
		}
		text = cleanJSON(text)
		var examples []asset.Example
		if err := json.Unmarshal([]byte(text), &examples); err != nil {
			return SuggestTechniqueResponse{}, fmt.Errorf("parse few-shot response: %w (raw: %.300s)", err, text)
		}
		merged.Examples = append(merged.Examples, examples...)

	case "role-priming":
		text, err := c.Generate(ctx, roleSystemPrompt, userPrompt)
		if err != nil {
			return SuggestTechniqueResponse{}, fmt.Errorf("suggest role-priming: %w", err)
		}
		merged.Persona = cleanJSON(text)

	case "constraints-first":
		text, err := c.Generate(ctx, constraintsSystemPrompt, userPrompt)
		if err != nil {
			return SuggestTechniqueResponse{}, fmt.Errorf("suggest constraints: %w", err)
		}
		merged.Constraints = cleanJSON(text)

	default:
		return SuggestTechniqueResponse{}, fmt.Errorf("unsupported technique: %q", req.TechniqueID)
	}

	return SuggestTechniqueResponse{Assets: merged}, nil
}

func buildTechniqueUserPrompt(a asset.Assets) string {
	parts := "Goal: " + orEmpty(a.Goal) + "\n"
	parts += "Context: " + orEmpty(a.Context) + "\n"
	if a.Persona != "" {
		parts += "Persona: " + a.Persona + "\n"
	}
	if a.Tone != "" {
		parts += "Tone: " + a.Tone + "\n"
	}
	if a.Constraints != "" {
		parts += "Existing constraints: " + a.Constraints + "\n"
	}
	return parts
}
