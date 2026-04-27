// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Seqera

//go:build integration

package webhook

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/seqeralabs/staticreg/pkg/db"
)

// TestBatchAdapter_WritesToDedicatedSchema confirms that an unqualified
// INSERT issued by BatchServiceAdapter resolves to the dedicated schema
// because the pool sets search_path on every connection.
func TestBatchAdapter_WritesToDedicatedSchema(t *testing.T) {
	for k, v := range map[string]string{
		"STATICREG_DB_HOST":     "127.0.0.1",
		"STATICREG_DB_PORT":     "5432",
		"STATICREG_DB_USER":     "staticreg",
		"STATICREG_DB_PASSWORD": "password",
		"STATICREG_DB_NAME":     "staticreg",
		"STATICREG_DB_SSLMODE":  "disable",
		"STATICREG_DB_SCHEMA":   "staticreg_batch_test",
	} {
		t.Setenv(k, v)
	}
	pool := db.InitPool()
	if pool == nil {
		t.Fatal("InitPool returned nil")
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DROP SCHEMA IF EXISTS staticreg_batch_test CASCADE")
		pool.Close()
	})

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	adapter := NewBatchServiceAdapter(log, pool)

	evt := &DistributionEvent{
		Timestamp: time.Now().UTC(),
		Action:    "pull",
		Target: DistributionEventTarget{
			MediaType:  "application/vnd.docker.distribution.manifest.v2+json",
			Repository: "example/app",
			Tag:        "v1.0.0",
			Digest:     "sha256:deadbeef",
		},
		Request: DistributionEventRequest{
			UserAgent: "docker/20.10.7 go/go1.16.4 kernel/5.10.0 os/linux arch/amd64",
		},
	}

	if err := adapter.SavePullEvent(context.Background(), evt); err != nil {
		t.Fatalf("SavePullEvent: %v", err)
	}

	// Allow the batch worker to flush.
	if err := adapter.Close(5 * time.Second); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM staticreg_batch_test.container_pull_metrics WHERE repo_name = 'example/app'`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row in dedicated schema, got %d", count)
	}
}
