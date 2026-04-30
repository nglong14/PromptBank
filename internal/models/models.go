package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Prompt struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Category  string    `json:"category"`
	Tags      []string  `json:"tags"`
	OwnerID   uuid.UUID `json:"ownerId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PromptVersion struct {
	ID              uuid.UUID        `json:"id"`
	PromptID        uuid.UUID        `json:"promptId"`
	VersionNumber   int              `json:"versionNumber"`
	Assets          json.RawMessage  `json:"assets"`
	FrameworkID     string           `json:"frameworkId"`
	TechniqueIDs    []string         `json:"techniqueIds"`
	ComposedOut     string           `json:"composedOutput"`
	DiffFromParent  *json.RawMessage `json:"diffFromParent,omitempty"`
	CreatedBy       *uuid.UUID       `json:"createdBy,omitempty"`
	ChangeType      string           `json:"changeType"`
	ChangeSummary   *string          `json:"changeSummary,omitempty"`
	ParentVersionID *uuid.UUID       `json:"parentVersionId,omitempty"`
	CreatedAt       time.Time        `json:"createdAt"`
}

type PromptLineage struct {
	PromptID             uuid.UUID  `json:"promptId"`
	DerivedFromPromptID  uuid.UUID  `json:"derivedFromPromptId"`
	DerivedFromVersionID *uuid.UUID `json:"derivedFromVersionId,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
}
