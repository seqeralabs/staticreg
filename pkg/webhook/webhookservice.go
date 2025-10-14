package webhook

import (
	"context"
)

// WebhookService defines the contract for all post-webhook operations.
type WebhookService interface {
	// Invalidation methods (from CacheManager)
	InvalidateRepository(repository string) error
	InvalidateAll() error

	// Database operation method (updated to use the concrete type)
	SavePullEvent(ctx context.Context, event *DistributionEvent) error
}
