-- 002_version_attribution: add change-attribution columns to prompt_versions.

ALTER TABLE prompt_versions
    ADD COLUMN created_by UUID REFERENCES users(id),
    ADD COLUMN change_type TEXT NOT NULL DEFAULT 'manual_edit'
        CHECK (change_type IN ('manual_edit','llm_refine','fork','bulk_update')),
    ADD COLUMN change_summary TEXT,
    ADD COLUMN parent_version_id UUID REFERENCES prompt_versions(id);

CREATE INDEX idx_versions_parent ON prompt_versions(parent_version_id);

-- Backfill linear chains: every v(n) in a prompt links to v(n-1) of the same prompt.
UPDATE prompt_versions pv
SET parent_version_id = parent.id
FROM prompt_versions parent
WHERE pv.prompt_id = parent.prompt_id
  AND parent.version_number = pv.version_number - 1;

-- Backfill forks from prompt_lineage: derived v1 rows point cross-prompt at the source version.
UPDATE prompt_versions pv
SET parent_version_id = pl.derived_from_version_id,
    change_type = 'fork'
FROM prompt_lineage pl
WHERE pv.prompt_id = pl.prompt_id
  AND pv.version_number = 1
  AND pl.derived_from_version_id IS NOT NULL;
