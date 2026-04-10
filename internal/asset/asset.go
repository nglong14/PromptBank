package asset

import (
	"encoding/json"
	"strings"
)

type Assets struct {
	Persona     string    `json:"persona"`
	Context     string    `json:"context"`
	Tone        string    `json:"tone"`
	Constraints string    `json:"constraints"`
	Examples    []Example `json:"examples"`
	Goal        string    `json:"goal"`
}

type Example struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type FieldQuality string

const (
	QualityEmpty    FieldQuality = "empty"
	QualityWeak     FieldQuality = "weak"
	QualityComplete FieldQuality = "complete"
)

const weakThreshold = 20

type NormalizedAssets struct {
	Assets      Assets                  `json:"assets"`
	FieldReport map[string]FieldQuality `json:"fieldReport"`
}

func ParseRaw(raw json.RawMessage) (Assets, error) {
	var a Assets
	if len(raw) == 0 || string(raw) == "{}" {
		return a, nil
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return a, err
	}
	return a, nil
}

func Normalize(a Assets) NormalizedAssets {
	a.Persona = strings.TrimSpace(a.Persona)
	a.Context = strings.TrimSpace(a.Context)
	a.Tone = strings.TrimSpace(a.Tone)
	a.Constraints = strings.TrimSpace(a.Constraints)
	a.Goal = strings.TrimSpace(a.Goal)

	for i := range a.Examples {
		a.Examples[i].Input = strings.TrimSpace(a.Examples[i].Input)
		a.Examples[i].Output = strings.TrimSpace(a.Examples[i].Output)
	}

	report := map[string]FieldQuality{
		"persona":     classifyText(a.Persona),
		"context":     classifyText(a.Context),
		"tone":        classifyText(a.Tone),
		"constraints": classifyText(a.Constraints),
		"goal":        classifyText(a.Goal),
		"examples":    classifyExamples(a.Examples),
	}

	return NormalizedAssets{Assets: a, FieldReport: report}
}

func (na NormalizedAssets) ToRawJSON() (json.RawMessage, error) {
	return json.Marshal(na.Assets)
}

func classifyText(s string) FieldQuality {
	if s == "" {
		return QualityEmpty
	}
	if len(s) < weakThreshold {
		return QualityWeak
	}
	return QualityComplete
}

func classifyExamples(examples []Example) FieldQuality {
	if len(examples) == 0 {
		return QualityEmpty
	}
	for _, ex := range examples {
		if ex.Input != "" && ex.Output != "" {
			return QualityComplete
		}
	}
	return QualityWeak
}
