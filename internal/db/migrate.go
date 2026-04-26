package db

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nglong14/PromptBank/db/migrations"
)

// initSchema is the bootstrap schema applied idempotently on every startup. It captures
// the state of migration 001_init plus the diff_from_parent column (which predates the
// versioned-migration runner). Versioned migrations under db/migrations/ run after it.
const initSchema = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS prompts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    category TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS prompt_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    prompt_id UUID NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    assets JSONB NOT NULL DEFAULT '{}',
    framework_id TEXT NOT NULL DEFAULT '',
    technique_ids TEXT[] NOT NULL DEFAULT '{}',
    composed_output TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(prompt_id, version_number)
);

CREATE TABLE IF NOT EXISTS prompt_lineage (
    id BIGSERIAL PRIMARY KEY,
    prompt_id UUID NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
    derived_from_prompt_id UUID NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
    derived_from_version_id UUID REFERENCES prompt_versions(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(prompt_id)
);

ALTER TABLE prompt_versions
  ADD COLUMN IF NOT EXISTS diff_from_parent JSONB;

CREATE INDEX IF NOT EXISTS idx_versions_diff_gin
  ON prompt_versions USING GIN (diff_from_parent);

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS prompts_set_updated_at ON prompts;
CREATE TRIGGER prompts_set_updated_at
BEFORE UPDATE ON prompts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
`

const initVersion = "001_init"

// ApplyMigrations brings the database to the current schema version by:
//  1. running the embedded bootstrap schema (idempotent),
//  2. ensuring the schema_migrations ledger exists and treating 001_init as already applied,
//  3. walking *.up.sql files from the embedded migrations FS in lexicographic order and
//     applying any not yet recorded, each inside its own transaction.
func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, initSchema); err != nil {
		return fmt.Errorf("run init schema: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`,
		initVersion,
	); err != nil {
		return fmt.Errorf("seed %s in schema_migrations: %w", initVersion, err)
	}

	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}

	versions := make([]string, 0, len(entries))
	files := make(map[string]string, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		version := strings.TrimSuffix(name, ".up.sql")
		versions = append(versions, version)
		files[version] = name
	}
	sort.Strings(versions)

	for _, version := range versions {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version,
		).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if exists {
			continue
		}

		body, err := migrations.FS.ReadFile(files[version])
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		if err := applyMigration(ctx, pool, version, string(body)); err != nil {
			return err
		}
	}

	return nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, version, body string) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration %s tx: %w", version, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, body); err != nil {
		return fmt.Errorf("apply migration %s: %w", version, err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`, version,
	); err != nil {
		return fmt.Errorf("record migration %s: %w", version, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", version, err)
	}
	return nil
}
