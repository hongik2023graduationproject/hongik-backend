-- Migration 002: Add search indexes for snippets
-- Improves performance of LOWER(title) LIKE and LOWER(description) LIKE queries

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_snippets_title_trgm
    ON snippets USING GIN (LOWER(title) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_snippets_description_trgm
    ON snippets USING GIN (LOWER(description) gin_trgm_ops);
