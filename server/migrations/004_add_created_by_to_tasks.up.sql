ALTER TABLE tasks ADD COLUMN created_by UUID NOT NULL REFERENCES users(id);

CREATE INDEX IF NOT EXISTS idx_tasks_created_by ON tasks(created_by);
