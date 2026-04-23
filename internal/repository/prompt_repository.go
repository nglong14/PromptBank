package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nglong14/PromptBank/internal/diff"
	"github.com/nglong14/PromptBank/internal/models"
)

type PromptRepository struct {
	pool *pgxpool.Pool
}

func NewPromptRepository(pool *pgxpool.Pool) *PromptRepository {
	return &PromptRepository{pool: pool}
}

type CreatePromptInput struct {
	Title    string
	Status   string
	Category string
	Tags     []string
	OwnerID  uuid.UUID
}

type CreateVersionInput struct {
	PromptID     uuid.UUID
	Assets       json.RawMessage
	FrameworkID  string
	TechniqueIDs []string
	ComposedOut  string
}

type DerivePromptInput struct {
	SourcePromptID  uuid.UUID
	SourceVersionID *uuid.UUID
	NewTitle        string
	OwnerID         uuid.UUID
}

func (r *PromptRepository) Create(ctx context.Context, in CreatePromptInput) (*models.Prompt, error) {
	var prompt models.Prompt
	err := r.pool.QueryRow(ctx, `
		INSERT INTO prompts (title, status, category, tags, owner_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, status, category, tags, owner_id, created_at, updated_at
	`, in.Title, in.Status, in.Category, in.Tags, in.OwnerID).Scan(
		&prompt.ID, &prompt.Title, &prompt.Status, &prompt.Category, &prompt.Tags,
		&prompt.OwnerID, &prompt.CreatedAt, &prompt.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create prompt: %w", err)
	}
	return &prompt, nil
}

func (r *PromptRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]models.Prompt, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, status, category, tags, owner_id, created_at, updated_at
		FROM prompts
		WHERE owner_id = $1
		ORDER BY updated_at DESC
	`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list prompts: %w", err)
	}
	defer rows.Close()

	prompts := make([]models.Prompt, 0)
	for rows.Next() {
		var p models.Prompt
		if err := rows.Scan(&p.ID, &p.Title, &p.Status, &p.Category, &p.Tags, &p.OwnerID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan prompt row: %w", err)
		}
		prompts = append(prompts, p)
	}
	return prompts, rows.Err()
}

func (r *PromptRepository) GetByID(ctx context.Context, id, ownerID uuid.UUID) (*models.Prompt, error) {
	var prompt models.Prompt
	err := r.pool.QueryRow(ctx, `
		SELECT id, title, status, category, tags, owner_id, created_at, updated_at
		FROM prompts
		WHERE id = $1 AND owner_id = $2
	`, id, ownerID).Scan(
		&prompt.ID, &prompt.Title, &prompt.Status, &prompt.Category, &prompt.Tags,
		&prompt.OwnerID, &prompt.CreatedAt, &prompt.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &prompt, nil
}

func (r *PromptRepository) Update(ctx context.Context, promptID, ownerID uuid.UUID, title, status, category string, tags []string) (*models.Prompt, error) {
	var prompt models.Prompt
	err := r.pool.QueryRow(ctx, `
		UPDATE prompts
		SET title = $3, status = $4, category = $5, tags = $6
		WHERE id = $1 AND owner_id = $2
		RETURNING id, title, status, category, tags, owner_id, created_at, updated_at
	`, promptID, ownerID, title, status, category, tags).Scan(
		&prompt.ID, &prompt.Title, &prompt.Status, &prompt.Category, &prompt.Tags,
		&prompt.OwnerID, &prompt.CreatedAt, &prompt.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &prompt, nil
}

func (r *PromptRepository) CreateVersion(ctx context.Context, in CreateVersionInput) (*models.PromptVersion, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	var versionNumber int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version_number), 0) + 1
		FROM prompt_versions
		WHERE prompt_id = $1
	`, in.PromptID).Scan(&versionNumber)
	if err != nil {
		return nil, fmt.Errorf("get version number: %w", err)
	}

	var version models.PromptVersion
	err = tx.QueryRow(ctx, `
		INSERT INTO prompt_versions (prompt_id, version_number, assets, framework_id, technique_ids, composed_output)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, prompt_id, version_number, assets, framework_id, technique_ids, composed_output, created_at
	`, in.PromptID, versionNumber, in.Assets, in.FrameworkID, in.TechniqueIDs, in.ComposedOut).Scan(
		&version.ID, &version.PromptID, &version.VersionNumber, &version.Assets, &version.FrameworkID,
		&version.TechniqueIDs, &version.ComposedOut, &version.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert version: %w", err)
	}

	// Compute and persist diff_from_parent for v2+. Failure is non-fatal: we log and continue so that a diff bug never blocks version creation.
	if versionNumber > 1 {
		var parent models.PromptVersion
		var parentDiffBytes []byte
		scanErr := tx.QueryRow(ctx, `
			SELECT id, prompt_id, version_number, assets, framework_id, technique_ids, composed_output, diff_from_parent, created_at
			FROM prompt_versions
			WHERE prompt_id = $1 AND version_number = $2
		`, in.PromptID, versionNumber-1).Scan(
			&parent.ID, &parent.PromptID, &parent.VersionNumber, &parent.Assets, &parent.FrameworkID,
			&parent.TechniqueIDs, &parent.ComposedOut, &parentDiffBytes, &parent.CreatedAt,
		)
		if parentDiffBytes != nil {
			raw := json.RawMessage(parentDiffBytes)
			parent.DiffFromParent = &raw
		}
		if scanErr != nil {
			log.Printf("diff: fetch parent version %d for prompt %s: %v", versionNumber-1, in.PromptID, scanErr)
		} else {
			diffResult, diffErr := diff.Compute(&parent, &version)
			if diffErr != nil {
				log.Printf("diff: compute for prompt %s v%d: %v", in.PromptID, versionNumber, diffErr)
			} else {
				diffJSON, marshalErr := json.Marshal(diffResult)
				if marshalErr != nil {
					log.Printf("diff: marshal result for prompt %s v%d: %v", in.PromptID, versionNumber, marshalErr)
				} else {
					if _, updateErr := tx.Exec(ctx,
						`UPDATE prompt_versions SET diff_from_parent = $1 WHERE id = $2`,
						diffJSON, version.ID,
					); updateErr != nil {
						log.Printf("diff: update diff_from_parent for prompt %s v%d: %v", in.PromptID, versionNumber, updateErr)
					} else {
						raw := json.RawMessage(diffJSON)
						version.DiffFromParent = &raw
					}
				}
			}
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE prompts SET updated_at = NOW() WHERE id = $1`, in.PromptID); err != nil {
		return nil, fmt.Errorf("touch prompt: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &version, nil
}

func (r *PromptRepository) ListVersions(ctx context.Context, promptID, ownerID uuid.UUID) ([]models.PromptVersion, error) {
	// owner check so users cannot list versions of another user's prompt
	if _, err := r.GetByID(ctx, promptID, ownerID); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, prompt_id, version_number, assets, framework_id, technique_ids, composed_output, diff_from_parent, created_at
		FROM prompt_versions
		WHERE prompt_id = $1
		ORDER BY version_number DESC
	`, promptID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	versions := make([]models.PromptVersion, 0)
	for rows.Next() {
		var v models.PromptVersion
		var diffBytes []byte
		if err := rows.Scan(&v.ID, &v.PromptID, &v.VersionNumber, &v.Assets, &v.FrameworkID, &v.TechniqueIDs, &v.ComposedOut, &diffBytes, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan version row: %w", err)
		}
		if diffBytes != nil {
			raw := json.RawMessage(diffBytes)
			v.DiffFromParent = &raw
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

// GetVersionByNumber fetches a single prompt version by its sequential number.
func (r *PromptRepository) GetVersionByNumber(ctx context.Context, promptID uuid.UUID, versionNumber int) (*models.PromptVersion, error) {
	var v models.PromptVersion
	var diffBytes []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, prompt_id, version_number, assets, framework_id, technique_ids, composed_output, diff_from_parent, created_at
		FROM prompt_versions
		WHERE prompt_id = $1 AND version_number = $2
	`, promptID, versionNumber).Scan(
		&v.ID, &v.PromptID, &v.VersionNumber, &v.Assets, &v.FrameworkID,
		&v.TechniqueIDs, &v.ComposedOut, &diffBytes, &v.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if diffBytes != nil {
		raw := json.RawMessage(diffBytes)
		v.DiffFromParent = &raw
	}
	return &v, nil
}

func (r *PromptRepository) DerivePrompt(ctx context.Context, in DerivePromptInput) (*models.Prompt, *models.PromptVersion, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	var sourceExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM prompts WHERE id = $1)
	`, in.SourcePromptID).Scan(&sourceExists); err != nil {
		return nil, nil, fmt.Errorf("check source prompt: %w", err)
	}
	if !sourceExists {
		return nil, nil, pgx.ErrNoRows
	}

	var sourceVersion models.PromptVersion
	var sourceDiffBytes []byte
	if in.SourceVersionID != nil {
		err = tx.QueryRow(ctx, `
			SELECT id, prompt_id, version_number, assets, framework_id, technique_ids, composed_output, diff_from_parent, created_at
			FROM prompt_versions
			WHERE id = $1 AND prompt_id = $2
		`, *in.SourceVersionID, in.SourcePromptID).Scan(
			&sourceVersion.ID, &sourceVersion.PromptID, &sourceVersion.VersionNumber, &sourceVersion.Assets,
			&sourceVersion.FrameworkID, &sourceVersion.TechniqueIDs, &sourceVersion.ComposedOut, &sourceDiffBytes, &sourceVersion.CreatedAt,
		)
	} else {
		err = tx.QueryRow(ctx, `
			SELECT id, prompt_id, version_number, assets, framework_id, technique_ids, composed_output, diff_from_parent, created_at
			FROM prompt_versions
			WHERE prompt_id = $1
			ORDER BY version_number DESC
			LIMIT 1
		`, in.SourcePromptID).Scan(
			&sourceVersion.ID, &sourceVersion.PromptID, &sourceVersion.VersionNumber, &sourceVersion.Assets,
			&sourceVersion.FrameworkID, &sourceVersion.TechniqueIDs, &sourceVersion.ComposedOut, &sourceDiffBytes, &sourceVersion.CreatedAt,
		)
	}
	if err != nil {
		return nil, nil, err
	}
	if sourceDiffBytes != nil {
		raw := json.RawMessage(sourceDiffBytes)
		sourceVersion.DiffFromParent = &raw
	}

	var derivedPrompt models.Prompt
	err = tx.QueryRow(ctx, `
		INSERT INTO prompts (title, status, category, tags, owner_id)
		SELECT $1, 'draft', category, tags, $2
		FROM prompts
		WHERE id = $3
		RETURNING id, title, status, category, tags, owner_id, created_at, updated_at
	`, in.NewTitle, in.OwnerID, in.SourcePromptID).Scan(
		&derivedPrompt.ID, &derivedPrompt.Title, &derivedPrompt.Status, &derivedPrompt.Category, &derivedPrompt.Tags,
		&derivedPrompt.OwnerID, &derivedPrompt.CreatedAt, &derivedPrompt.UpdatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create derived prompt: %w", err)
	}

	var derivedVersion models.PromptVersion
	err = tx.QueryRow(ctx, `
		INSERT INTO prompt_versions (prompt_id, version_number, assets, framework_id, technique_ids, composed_output)
		VALUES ($1, 1, $2, $3, $4, $5)
		RETURNING id, prompt_id, version_number, assets, framework_id, technique_ids, composed_output, created_at
	`, derivedPrompt.ID, sourceVersion.Assets, sourceVersion.FrameworkID, sourceVersion.TechniqueIDs, sourceVersion.ComposedOut).Scan(
		&derivedVersion.ID, &derivedVersion.PromptID, &derivedVersion.VersionNumber, &derivedVersion.Assets,
		&derivedVersion.FrameworkID, &derivedVersion.TechniqueIDs, &derivedVersion.ComposedOut, &derivedVersion.CreatedAt,
	)
	// diff_from_parent is intentionally NULL for derived v1 rows.
	if err != nil {
		return nil, nil, fmt.Errorf("create derived version: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO prompt_lineage (prompt_id, derived_from_prompt_id, derived_from_version_id)
		VALUES ($1, $2, $3)
	`, derivedPrompt.ID, in.SourcePromptID, sourceVersion.ID); err != nil {
		return nil, nil, fmt.Errorf("create lineage row: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit derive tx: %w", err)
	}

	return &derivedPrompt, &derivedVersion, nil
}
