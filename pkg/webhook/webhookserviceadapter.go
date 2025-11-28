package webhook

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ServiceAdapter struct {
	Logger *slog.Logger
	Pool   *pgxpool.Pool
}

func NewServiceAdapter(log *slog.Logger, pool *pgxpool.Pool) *ServiceAdapter {
	return &ServiceAdapter{
		Logger: log,
		Pool:   pool,
	}
}

// Close is a no-op for synchronous ServiceAdapter (implements WebhookService interface)
func (a *ServiceAdapter) Close(timeout time.Duration) error {
	return nil
}

func (a *ServiceAdapter) SavePullEvent(ctx context.Context, event *DistributionEvent) error {
	if a.Pool == nil {
		a.Logger.Debug("Skipping pull event save: Database pool is not active.")
		return nil
	}

	repoName := event.Target.Repository
	tag := event.Target.Tag
	digest := event.Target.Digest
	architecture := event.GetArchitecture()
	pullDate := event.Timestamp.UTC().Format("2006-01-02") // Format as DATE (YYYY-MM-DD) in UTC

	// Use UPSERT (INSERT ... ON CONFLICT) to increment pull count
	// If the combination of (pull_date, repo_name, tag, digest, architecture) exists, increment pull_count
	// Otherwise, insert a new record with pull_count = 1
	query := `
        INSERT INTO container_pull_metrics (pull_date, repo_name, tag, digest, architecture, pull_count, updated_at)
        VALUES ($1, $2, $3, $4, $5, 1, NOW())
        ON CONFLICT (pull_date, repo_name, tag, digest, architecture)
        DO UPDATE SET
            pull_count = container_pull_metrics.pull_count + 1,
            updated_at = NOW()`

	_, err := a.Pool.Exec(ctx, query,
		pullDate,
		repoName,
		tag,
		digest,
		architecture,
	)

	if err != nil {
		return fmt.Errorf("failed to execute upsert query for pull metrics: %w", err)
	}

	return nil
}
