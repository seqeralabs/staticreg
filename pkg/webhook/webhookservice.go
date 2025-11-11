package webhook

import (
	"context"
)

// WebhookService defines the contract for all post-webhook operations.
type WebhookService interface {
	// Database operation method
	SavePullEvent(ctx context.Context, event *DistributionEvent) error
}
