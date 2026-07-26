CREATE TYPE difficulty_level AS ENUM ('easy', 'medium', 'hard');

CREATE TABLE IF NOT EXISTS tasks (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug             TEXT NOT NULL UNIQUE,
    title            TEXT NOT NULL,
    statement        TEXT NOT NULL,
    difficulty       difficulty_level NOT NULL,

    time_limit_ms    INTEGER NOT NULL DEFAULT 1000,
    memory_limit_mb  INTEGER NOT NULL DEFAULT 256,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tasks_difficulty ON tasks(difficulty);