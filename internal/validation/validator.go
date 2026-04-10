package validation

import (
	"fmt"

	"github.com/nglong14/PromptBank/internal/asset"
	"github.com/nglong14/PromptBank/internal/framework"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Diagnostic struct {
	SlotName string   `json:"slotName"`
	Field    string   `json:"field"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

func ValidateSlots(f framework.Framework, normalized asset.NormalizedAssets) []Diagnostic {
	var diags []Diagnostic

	for _, slot := range f.Slots {
		quality, ok := normalized.FieldReport[slot.AssetField]
		if !ok {
			quality = asset.QualityEmpty
		}

		switch {
		case slot.Required && quality == asset.QualityEmpty:
			diags = append(diags, Diagnostic{
				SlotName: slot.Name,
				Field:    slot.AssetField,
				Severity: SeverityError,
				Message:  fmt.Sprintf("Required slot %q mapped to asset field %q is empty", slot.Name, slot.AssetField),
			})
		case slot.Required && quality == asset.QualityWeak:
			diags = append(diags, Diagnostic{
				SlotName: slot.Name,
				Field:    slot.AssetField,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Required slot %q mapped to asset field %q is weak (too short)", slot.Name, slot.AssetField),
			})
		case !slot.Required && quality == asset.QualityEmpty:
			diags = append(diags, Diagnostic{
				SlotName: slot.Name,
				Field:    slot.AssetField,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Optional slot %q mapped to asset field %q is empty", slot.Name, slot.AssetField),
			})
		}
	}

	return diags
}

func HasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}
