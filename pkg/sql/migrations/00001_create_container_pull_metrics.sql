-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS container_pull_metrics (
    id BIGSERIAL PRIMARY KEY,
    pull_date DATE NOT NULL,
    repo_name TEXT NOT NULL,
    tag TEXT NOT NULL,
    digest TEXT NOT NULL,
    architecture TEXT NOT NULL,
    pull_count INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(pull_date, repo_name, tag, digest, architecture)
);

CREATE INDEX IF NOT EXISTS idx_pull_date ON container_pull_metrics (pull_date DESC);
CREATE INDEX IF NOT EXISTS idx_repo_date ON container_pull_metrics (repo_name, pull_date DESC);
CREATE INDEX IF NOT EXISTS idx_repo_arch_date ON container_pull_metrics (repo_name, architecture, pull_date DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_repo_arch_date;
DROP INDEX IF EXISTS idx_repo_date;
DROP INDEX IF EXISTS idx_pull_date;
DROP TABLE IF EXISTS container_pull_metrics;
-- +goose StatementEnd
