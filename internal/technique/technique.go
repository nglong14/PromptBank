package technique

import (
	"fmt"
	"strings"

	"github.com/nglong14/PromptBank/internal/asset"
)

type Technique struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var registry []Technique

func init() {
	registry = []Technique{
		{
			ID:          "few-shot",
			Name:        "Few-Shot Examples",
			Description: "Inject input/output examples before the main prompt so the model learns the pattern.",
		},
		{
			ID:          "role-priming",
			Name:        "Role Priming",
			Description: "Prepend a persona block that sets the AI's identity and expertise.",
		},
		{
			ID:          "constraints-first",
			Name:        "Constraints First",
			Description: "Lead with constraints and rules before the main task to bound the output.",
		},
	}
}

func List() []Technique {
	out := make([]Technique, len(registry))
	copy(out, registry)
	return out
}

func Get(id string) (Technique, bool) {
	for _, t := range registry {
		if t.ID == id {
			return t, true
		}
	}
	return Technique{}, false
}

func Apply(ids []string, assets asset.Assets, body string) string {
	var parts []string

	for _, id := range ids {
		switch id {
		case "role-priming":
			if assets.Persona != "" {
				parts = append(parts, fmt.Sprintf("[ROLE]\nYou are %s.\n", assets.Persona))
			}
		case "constraints-first":
			if assets.Constraints != "" {
				parts = append(parts, fmt.Sprintf("[CONSTRAINTS]\n%s\n", assets.Constraints))
			}
		case "few-shot":
			if len(assets.Examples) > 0 {
				var exBuf strings.Builder
				exBuf.WriteString("[EXAMPLES]\n")
				for i, ex := range assets.Examples {
					if ex.Input == "" && ex.Output == "" {
						continue
					}
					exBuf.WriteString(fmt.Sprintf("Example %d:\n  Input: %s\n  Output: %s\n", i+1, ex.Input, ex.Output))
				}
				parts = append(parts, exBuf.String())
			}
		}
	}

	parts = append(parts, body)
	return strings.Join(parts, "\n")
}
