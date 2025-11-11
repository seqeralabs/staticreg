package webhook

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/seqeralabs/staticreg/pkg/db"
)

func TestNewServiceAdapter(t *testing.T) {
	logger := slog.Default()

	adapter := NewServiceAdapter(logger)

	if adapter == nil {
		t.Fatal("Expected adapter to be initialized, got nil")
	}
	if adapter.Logger != logger {
		t.Error("Logger was not correctly assigned")
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

	originalPool := db.Pool

	defer func() { db.Pool = originalPool }()

	t.Run("DB_Disconnected_Graceful_Skip", func(t *testing.T) {
		db.Pool = nil
		adapter := NewServiceAdapter(slog.Default())

		err := adapter.SavePullEvent(ctx, &testEvent)

		if err != nil {
			t.Errorf("Expected nil error on skip, got %v", err)
		}
	})
}
