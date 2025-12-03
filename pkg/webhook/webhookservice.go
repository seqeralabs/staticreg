package webhook

import (
	"context"
	"time"
)

// WebhookService defines the contract for all post-webhook operations.
type WebhookService interface {
	// Database operation method
	SavePullEvent(ctx context.Context, event *DistributionEvent) error
	// Close gracefully shuts down the service, flushing pending events
	Close(timeout time.Duration) error
}
