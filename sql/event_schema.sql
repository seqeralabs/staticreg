CREATE TABLE IF NOT EXISTS docker_pull_events (
                                                  id BIGSERIAL PRIMARY KEY,
                                                  event_time TIMESTAMP WITH TIME ZONE NOT NULL,
                                                  repo_name TEXT NOT NULL,
                                                  tag TEXT NOT NULL,
                                                  actor_name TEXT,
                                                  event_payload JSONB
);

CREATE INDEX IF NOT EXISTS idx_repo_time ON docker_pull_events (repo_name, event_time DESC);
