-- 002_version_attribution rollback. Leaves diff_from_parent in place (pre-existing).

DROP INDEX IF EXISTS idx_versions_parent;

ALTER TABLE prompt_versions
    DROP COLUMN IF EXISTS parent_version_id,
    DROP COLUMN IF EXISTS change_summary,
    DROP COLUMN IF EXISTS change_type,
    DROP COLUMN IF EXISTS created_by;
