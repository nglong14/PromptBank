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
	PromptID        uuid.UUID
	Assets          json.RawMessage
	FrameworkID     string
	TechniqueIDs    []string
	ComposedOut     string
	CreatedBy       uuid.UUID
	ChangeType      string
	ChangeSummary   *string
	ParentVersionID *uuid.UUID
}

// allowedChangeTypes mirrors the CHECK constraint on prompt_versions.change_type.
// The repo enforces it as defense-in-depth alongside the DB; callers that pass an
// unknown value get a clean Go error instead of a constraint violation.
var allowedChangeTypes = map[string]struct{}{
	"manual_edit": {},
	"llm_refine":  {},
	"fork":        {},
	"bulk_update": {},
}

// versionColumns lists the prompt_versions columns returned by every read path. Kept
// as a single constant so SELECT, INSERT RETURNING, and Scan ordering can never drift.
const versionColumns = `id, prompt_id, version_number, assets, framework_id, technique_ids, composed_output,
	diff_from_parent, created_by, change_type, change_summary, parent_version_id, created_at`

// scanVersion reads a row produced by a SELECT/RETURNING that uses versionColumns into v.
// row must be a pgx.Row (e.g. tx.QueryRow(...)) or a pgx.Rows positioned at a row.
func scanVersion(row pgx.Row, v *models.PromptVersion) error {
	var diffBytes []byte
	if err := row.Scan(
		&v.ID, &v.PromptID, &v.VersionNumber, &v.Assets, &v.FrameworkID,
		&v.TechniqueIDs, &v.ComposedOut, &diffBytes,
		&v.CreatedBy, &v.ChangeType, &v.ChangeSummary, &v.ParentVersionID,
		&v.CreatedAt,
	); err != nil {
		return err
	}
	if diffBytes != nil {
		raw := json.RawMessage(diffBytes)
		v.DiffFromParent = &raw
	}
	return nil
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
	if in.ChangeType == "" {
		in.ChangeType = "manual_edit"
	}
	if _, ok := allowedChangeTypes[in.ChangeType]; !ok {
		return nil, fmt.Errorf("invalid change_type %q", in.ChangeType)
	}

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

	// Resolve canonical parent: caller-supplied override wins; otherwise the latest
	// existing version of this prompt becomes the parent. v1 of a brand-new prompt has
	// no parent (the column stays NULL and no diff is computed).
	parentVersionID := in.ParentVersionID
	if parentVersionID == nil && versionNumber > 1 {
		var pid uuid.UUID
		if err := tx.QueryRow(ctx, `
			SELECT id FROM prompt_versions
			WHERE prompt_id = $1
			ORDER BY version_number DESC
			LIMIT 1
		`, in.PromptID).Scan(&pid); err != nil {
			return nil, fmt.Errorf("resolve parent version: %w", err)
		}
		parentVersionID = &pid
	}

	// Convert uuid.Nil to a real NULL on the wire. pgx writes a non-pointer uuid.Nil
	// as the zero UUID literal, which would violate the FK to users(id).
	var createdByParam *uuid.UUID
	if in.CreatedBy != uuid.Nil {
		createdByParam = &in.CreatedBy
	}

	var version models.PromptVersion
	if err := scanVersion(tx.QueryRow(ctx, `
		INSERT INTO prompt_versions (
			prompt_id, version_number, assets, framework_id, technique_ids, composed_output,
			created_by, change_type, change_summary, parent_version_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING `+versionColumns,
		in.PromptID, versionNumber, in.Assets, in.FrameworkID, in.TechniqueIDs, in.ComposedOut,
		createdByParam, in.ChangeType, in.ChangeSummary, parentVersionID,
	), &version); err != nil {
		return nil, fmt.Errorf("insert version: %w", err)
	}

	// Compute and persist diff_from_parent against the resolved parent. Failure is
	// non-fatal so a diff bug never blocks version creation.
	if parentVersionID != nil {
		var parent models.PromptVersion
		scanErr := scanVersion(tx.QueryRow(ctx, `
			SELECT `+versionColumns+`
			FROM prompt_versions
			WHERE id = $1
		`, *parentVersionID), &parent)
		if scanErr != nil {
			log.Printf("diff: fetch parent version %s for prompt %s: %v", *parentVersionID, in.PromptID, scanErr)
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
		SELECT `+versionColumns+`
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
		if err := scanVersion(rows, &v); err != nil {
			return nil, fmt.Errorf("scan version row: %w", err)
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

// GetVersionByNumber fetches a single prompt version by its sequential number.
func (r *PromptRepository) GetVersionByNumber(ctx context.Context, promptID uuid.UUID, versionNumber int) (*models.PromptVersion, error) {
	var v models.PromptVersion
	if err := scanVersion(r.pool.QueryRow(ctx, `
		SELECT `+versionColumns+`
		FROM prompt_versions
		WHERE prompt_id = $1 AND version_number = $2
	`, promptID, versionNumber), &v); err != nil {
		return nil, err
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
	if in.SourceVersionID != nil {
		err = scanVersion(tx.QueryRow(ctx, `
			SELECT `+versionColumns+`
			FROM prompt_versions
			WHERE id = $1 AND prompt_id = $2
		`, *in.SourceVersionID, in.SourcePromptID), &sourceVersion)
	} else {
		err = scanVersion(tx.QueryRow(ctx, `
			SELECT `+versionColumns+`
			FROM prompt_versions
			WHERE prompt_id = $1
			ORDER BY version_number DESC
			LIMIT 1
		`, in.SourcePromptID), &sourceVersion)
	}
	if err != nil {
		return nil, nil, err
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

	// The derived v1 carries fork attribution: created_by from the new owner, change_type
	// 'fork', and a cross-prompt parent_version_id pointing back at the source version.
	// diff_from_parent is intentionally NULL for derived v1 rows (the content is a copy).
	var derivedVersion models.PromptVersion
	if err := scanVersion(tx.QueryRow(ctx, `
		INSERT INTO prompt_versions (
			prompt_id, version_number, assets, framework_id, technique_ids, composed_output,
			created_by, change_type, parent_version_id
		)
		VALUES ($1, 1, $2, $3, $4, $5, $6, 'fork', $7)
		RETURNING `+versionColumns,
		derivedPrompt.ID, sourceVersion.Assets, sourceVersion.FrameworkID, sourceVersion.TechniqueIDs, sourceVersion.ComposedOut,
		in.OwnerID, sourceVersion.ID,
	), &derivedVersion); err != nil {
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
