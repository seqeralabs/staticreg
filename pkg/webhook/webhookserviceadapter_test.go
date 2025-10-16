package webhook

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgconn"
	"github.com/seqeralabs/staticreg/pkg/db"
)

type MockDBPool struct {
	execFunc func(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
}

func (m *MockDBPool) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, query, args...)
	}
	return pgconn.CommandTag{}, nil
}

type MockCacheManager struct {
	InvalidateRepositoryFunc func(repository string) error
}

func (m *MockCacheManager) InvalidateRepository(repository string) error {
	if m.InvalidateRepositoryFunc != nil {
		return m.InvalidateRepositoryFunc(repository)
	}
	return nil
}

func TestNewServiceAdapter(t *testing.T) {
	mockCM := &MockCacheManager{}
	logger := slog.Default()

	adapter := NewServiceAdapter(mockCM, logger)

	if adapter == nil {
		t.Fatal("Expected adapter to be initialized, got nil")
	}
	if adapter.CacheManager != mockCM {
		t.Error("CacheManager was not correctly assigned")
	}
	if adapter.Logger != logger {
		t.Error("Logger was not correctly assigned")
	}
}

func TestInvalidateRepository(t *testing.T) {
	testRepo := "test/repo"
	expectedErr := errors.New("cache invalidation failed")

	t.Run("Delegation_Success", func(t *testing.T) {
		mockCM := &MockCacheManager{
			InvalidateRepositoryFunc: func(repository string) error {
				if repository != testRepo {
					t.Errorf("Expected repository '%s', got '%s'", testRepo, repository)
				}
				return nil
			},
		}
		adapter := NewServiceAdapter(mockCM, slog.Default())
		err := adapter.InvalidateRepository(testRepo)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	})

	t.Run("Delegation_Failure", func(t *testing.T) {
		mockCM := &MockCacheManager{
			InvalidateRepositoryFunc: func(repository string) error {
				return expectedErr
			},
		}
		adapter := NewServiceAdapter(mockCM, slog.Default())
		err := adapter.InvalidateRepository(testRepo)
		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
		}
	})
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
		Actor: DistributionEventActor{
			Name: "test-user",
		},
	}

	originalPool := db.Pool

	defer func() { db.Pool = originalPool }()

	t.Run("DB_Disconnected_Graceful_Skip", func(t *testing.T) {
		db.Pool = nil
		adapter := NewServiceAdapter(nil, slog.Default())

		err := adapter.SavePullEvent(ctx, &testEvent)

		if err != nil {
			t.Errorf("Expected nil error on skip, got %v", err)
		}
	})
}
