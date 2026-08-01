CREATE TABLE IF NOT EXISTS task_examples (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id        UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    example_input  TEXT NOT NULL DEFAULT '',
    example_output TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_examples_task_id ON task_examples(task_id);
