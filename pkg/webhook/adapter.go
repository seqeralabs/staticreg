package webhook

import (
	"context"
	"fmt"
	"log/slog"
	_ "time"

	"github.com/goccy/go-json"
	"github.com/seqeralabs/staticreg/pkg/db" // Import your DB package
)

// ServiceAdapter implements the WebhookService by utilizing CacheManager and DB.
type ServiceAdapter struct {
	Logger *slog.Logger
}

// SavePullEvent is the new function to save the event to the PostgreSQL database.
func (a *ServiceAdapter) SavePullEvent(ctx context.Context, event *DistributionEvent) error {
	// 1. Check if DB is initialized (to respect the optional DB requirement)
	if db.Pool == nil {
		a.Logger.Debug("Skipping pull event save: Database pool is not active.")
		return nil // Return nil so the webhook doesn't fail
	}

	// 2. Prepare data for insertion (assuming event has required fields)
	repoName := event.Target.Repository
	tag := event.Target.Tag
	actorName := event.Actor.Name
	eventTime := event.Timestamp

	// We need the raw JSON payload to save as JSONB.
	// Since we don't have the original raw body here, we'll serialize the event struct.
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
