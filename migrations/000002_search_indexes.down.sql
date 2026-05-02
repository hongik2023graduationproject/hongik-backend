DROP INDEX IF EXISTS idx_snippets_description_trgm;
DROP INDEX IF EXISTS idx_snippets_title_trgm;

-- Intentionally not dropping the pg_trgm extension: other databases or
-- schemas in the cluster may depend on it. Leaving the extension in place
-- on rollback is the conservative choice.
