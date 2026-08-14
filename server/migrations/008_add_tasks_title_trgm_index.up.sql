CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_tasks_title_trgm
    ON tasks USING gin (lower(title) gin_trgm_ops);
