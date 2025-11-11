package webhook

import (
	"context"
	"fmt"
	"github.com/seqeralabs/staticreg/pkg/db"
	"log/slog"
)

type ServiceAdapter struct {
	Logger *slog.Logger
}

func NewServiceAdapter(log *slog.Logger) *ServiceAdapter {
	return &ServiceAdapter{
		Logger: log,
	}
}

func (a *ServiceAdapter) SavePullEvent(ctx context.Context, event *DistributionEvent) error {
	if db.Pool == nil {
		a.Logger.Debug("Skipping pull event save: Database pool is not active.")
		return nil
	}

	repoName := event.Target.Repository
	tag := event.Target.Tag
	digest := event.Target.Digest
	architecture := event.GetArchitecture()
	pullDate := event.Timestamp.Format("2006-01-02") // Format as DATE (YYYY-MM-DD)

	a.Logger.Debug("Processing pull event",
		"repository", repoName,
		"tag", tag,
		"digest", digest,
		"architecture", architecture,
		"date", pullDate)

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

	_, err := db.Pool.Exec(ctx, query,
		pullDate,
		repoName,
		tag,
		digest,
		architecture,
	)

	if err != nil {
		return fmt.Errorf("failed to execute upsert query for pull metrics: %w", err)
	}

	a.Logger.Info("Pull metrics updated",
		"repository", repoName,
		"tag", tag,
		"digest", digest,
		"architecture", architecture)

	return nil
}
