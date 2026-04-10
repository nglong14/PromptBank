package framework

type Slot struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	AssetField  string `json:"assetField"`
}

type Framework struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Slots       []Slot `json:"slots"`
}

type SlotMapping struct {
	SlotName string `json:"slotName"`
	Value    string `json:"value"`
	Source   string `json:"source"`
}

var catalog []Framework

func init() {
	catalog = []Framework{
		{
			ID:          "aida",
			Name:        "AIDA",
			Description: "Attention, Interest, Desire, Action — guides the reader through a persuasive sequence.",
			Slots: []Slot{
				{Name: "attention", Description: "Hook that grabs the reader's focus", Required: true, AssetField: "goal"},
				{Name: "interest", Description: "Context or details that build curiosity", Required: true, AssetField: "context"},
				{Name: "desire", Description: "Emotional or value-driven appeal", Required: true, AssetField: "tone"},
				{Name: "action", Description: "Clear instruction on what to do next", Required: true, AssetField: "constraints"},
			},
		},
		{
			ID:          "pas",
			Name:        "PAS",
			Description: "Problem, Agitation, Solution — identifies a pain point then resolves it.",
			Slots: []Slot{
				{Name: "problem", Description: "The core problem to address", Required: true, AssetField: "context"},
				{Name: "agitation", Description: "Why the problem matters or hurts", Required: true, AssetField: "tone"},
				{Name: "solution", Description: "The proposed resolution or output", Required: true, AssetField: "goal"},
			},
		},
		{
			ID:          "react",
			Name:        "ReAct-style",
			Description: "Role, Reasoning, Action, Constraint — structured for task-oriented agents.",
			Slots: []Slot{
				{Name: "role", Description: "Who the AI should act as", Required: true, AssetField: "persona"},
				{Name: "reasoning", Description: "Background context and reasoning chain", Required: false, AssetField: "context"},
				{Name: "action", Description: "What the AI should produce", Required: true, AssetField: "goal"},
				{Name: "constraint", Description: "Rules and boundaries", Required: true, AssetField: "constraints"},
			},
		},
	}
}

func List() []Framework {
	out := make([]Framework, len(catalog))
	copy(out, catalog)
	return out
}

func Get(id string) (Framework, bool) {
	for _, f := range catalog {
		if f.ID == id {
			return f, true
		}
	}
	return Framework{}, false
}

func MapSlots(f Framework, assetFields map[string]string) []SlotMapping {
	mappings := make([]SlotMapping, 0, len(f.Slots))
	for _, slot := range f.Slots {
		value := assetFields[slot.AssetField]
		source := slot.AssetField
		if value == "" {
			source = ""
		}
		mappings = append(mappings, SlotMapping{
			SlotName: slot.Name,
			Value:    value,
			Source:   source,
		})
	}
	return mappings
}
