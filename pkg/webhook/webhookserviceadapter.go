package webhook

import (
	"context"
	"log/slog"

	"github.com/seqeralabs/staticreg/pkg/metrics"
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
	repoName := event.Target.Repository
	tag := event.Target.Tag
	digest := event.Target.Digest
	architecture := event.GetArchitecture()

	// Record the pull event to Prometheus metrics
	metrics.RecordPull(repoName, tag, digest, architecture)

	a.Logger.Debug("Recorded pull event to Prometheus",
		"repository", repoName,
		"tag", tag,
		"digest", digest,
		"architecture", architecture)

	return nil
}
