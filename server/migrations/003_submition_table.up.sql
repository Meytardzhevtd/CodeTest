CREATE TABLE IF NOT EXISTS submissions (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id          UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    code             TEXT NOT NULL,
    language         TEXT NOT NULL,

    status           TEXT NOT NULL DEFAULT 'pending',
    output           TEXT,
    error            TEXT,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submissions_task_id ON submissions(task_id);
CREATE INDEX IF NOT EXISTS idx_submissions_user_id ON submissions(user_id);
CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
CREATE INDEX IF NOT EXISTS idx_submissions_created_at ON submissions(created_at DESC);