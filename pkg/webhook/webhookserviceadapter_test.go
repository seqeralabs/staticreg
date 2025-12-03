package webhook

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestNewServiceAdapter(t *testing.T) {
	logger := slog.Default()

	adapter := NewServiceAdapter(logger, nil)

	if adapter == nil {
		t.Fatal("Expected adapter to be initialized, got nil")
	}
	if adapter.Logger != logger {
		t.Error("Logger was not correctly assigned")
	}
	if adapter.Pool != nil {
		t.Error("Expected Pool to be nil")
	}
}

func TestSavePullEvent(t *testing.T) {
	ctx := context.Background()

	testEvent := DistributionEvent{
		Timestamp: time.Now(),
		Action:    "pull",
		Target: DistributionEventTarget{
			Repository: "my-image",
			Tag:        "v1.0",
		},
		Request: DistributionEventRequest{
			UserAgent: "docker/20.10.7 go/go1.16.4 os/linux arch/amd64",
		},
	}

	t.Run("DB_Disconnected_Graceful_Skip", func(t *testing.T) {
		adapter := NewServiceAdapter(slog.Default(), nil)

		err := adapter.SavePullEvent(ctx, &testEvent)

		if err != nil {
			t.Errorf("Expected nil error on skip, got %v", err)
		}
	})
}
