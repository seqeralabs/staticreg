package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/seqeralabs/staticreg/pkg/db"
	"log/slog"
)

type CacheInvalidator interface {
	InvalidateRepository(repository string) error
}

type ServiceAdapter struct {
	CacheManager CacheInvalidator
	Logger       *slog.Logger
}

func NewServiceAdapter(cm CacheInvalidator, log *slog.Logger) *ServiceAdapter {
	return &ServiceAdapter{
		CacheManager: cm,
		Logger:       log,
	}
}

func (a *ServiceAdapter) InvalidateRepository(repository string) error {
	return a.CacheManager.InvalidateRepository(repository)
}

func (a *ServiceAdapter) SavePullEvent(ctx context.Context, event *DistributionEvent) error {
	if db.Pool == nil {
		a.Logger.Debug("Skipping pull event save: Database pool is not active.")
		return nil
	}

	repoName := event.Target.Repository
	tag := event.Target.Tag
	actorName := event.Actor.Name
	eventTime := event.Timestamp

	jsonPayload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("could not marshal event for JSONB storage: %w", err)
	}

	query := `
        INSERT INTO docker_pull_events (event_time, repo_name, tag, actor_name, event_payload)
        VALUES ($1, $2, $3, $4, $5)`

	_, err = db.Pool.Exec(ctx, query,
		eventTime,
		repoName,
		tag,
		actorName,
		jsonPayload,
	)

	if err != nil {
		return fmt.Errorf("failed to execute insert query for pull event: %w", err)
	}

	return nil
}
