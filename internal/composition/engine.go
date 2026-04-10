package composition

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nglong14/PromptBank/internal/asset"
	"github.com/nglong14/PromptBank/internal/framework"
	"github.com/nglong14/PromptBank/internal/technique"
	"github.com/nglong14/PromptBank/internal/validation"
)

type ComposeInput struct {
	AssetsRaw    json.RawMessage `json:"assets"`
	FrameworkID  string          `json:"frameworkId"`
	TechniqueIDs []string        `json:"techniqueIds"`
}

type ComposeOutput struct {
	ComposedOutput string                  `json:"composedOutput"`
	SlotMap        []framework.SlotMapping `json:"slotMap"`
	Diagnostics    []validation.Diagnostic `json:"diagnostics"`
	FrameworkID    string                  `json:"frameworkId"`
	TechniqueIDs   []string                `json:"techniqueIds"`
}

func Compose(input ComposeInput) (ComposeOutput, error) {
	parsed, err := asset.ParseRaw(input.AssetsRaw)
	if err != nil {
		return ComposeOutput{}, fmt.Errorf("parse assets: %w", err)
	}

	normalized := asset.Normalize(parsed)

	fw, ok := framework.Get(input.FrameworkID)
	if !ok {
		return ComposeOutput{}, fmt.Errorf("unknown framework: %s", input.FrameworkID)
	}

	for _, tid := range input.TechniqueIDs {
		if _, found := technique.Get(tid); !found {
			return ComposeOutput{}, fmt.Errorf("unknown technique: %s", tid)
		}
	}

	assetFields := map[string]string{
		"persona":     normalized.Assets.Persona,
		"context":     normalized.Assets.Context,
		"tone":        normalized.Assets.Tone,
		"constraints": normalized.Assets.Constraints,
		"goal":        normalized.Assets.Goal,
	}

	slotMap := framework.MapSlots(fw, assetFields)
	diags := validation.ValidateSlots(fw, normalized)

	body := renderFrameworkBody(fw, slotMap)
	composed := technique.Apply(input.TechniqueIDs, normalized.Assets, body)

	return ComposeOutput{
		ComposedOutput: composed,
		SlotMap:        slotMap,
		Diagnostics:    diags,
		FrameworkID:    input.FrameworkID,
		TechniqueIDs:   input.TechniqueIDs,
	}, nil
}

func renderFrameworkBody(fw framework.Framework, mappings []framework.SlotMapping) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("[%s]\n", fw.Name))
	for _, m := range mappings {
		if m.Value == "" {
			b.WriteString(fmt.Sprintf("## %s\n(empty)\n\n", m.SlotName))
		} else {
			b.WriteString(fmt.Sprintf("## %s\n%s\n\n", m.SlotName, m.Value))
		}
	}
	return b.String()
}
