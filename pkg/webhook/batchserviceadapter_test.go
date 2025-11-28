// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Seqera
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package webhook

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestNewBatchServiceAdapter(t *testing.T) {
	logger := slog.Default()

	adapter := NewBatchServiceAdapter(logger, nil)

	if adapter == nil {
		t.Fatal("Expected adapter to be initialized, got nil")
	}
	if adapter.Logger != logger {
		t.Error("Logger was not correctly assigned")
	}
	if adapter.Pool != nil {
		t.Error("Expected Pool to be nil")
	}
	if adapter.eventChan == nil {
		t.Error("Expected eventChan to be initialized")
	}
	if adapter.batchSize <= 0 {
		t.Error("Expected batchSize to be positive")
	}
	if adapter.flushInterval <= 0 {
		t.Error("Expected flushInterval to be positive")
	}
}

func TestBatchedSavePullEvent_DBDisconnected(t *testing.T) {
	ctx := context.Background()
	adapter := NewBatchServiceAdapter(slog.Default(), nil)

	testEvent := DistributionEvent{
		Timestamp: time.Now(),
		Action:    "pull",
		Target: DistributionEventTarget{
			Repository: "my-image",
			Tag:        "v1.0",
			Digest:     "sha256:abc123",
		},
		Request: DistributionEventRequest{
			UserAgent: "docker/20.10.7 go/go1.16.4 os/linux arch/amd64",
		},
	}

	err := adapter.SavePullEvent(ctx, &testEvent)

	if err != nil {
		t.Errorf("Expected nil error on skip, got %v", err)
	}
}

func TestBatchedSavePullEvent_NonBlocking(t *testing.T) {
	// This test ensures SavePullEvent returns immediately without blocking
	adapter := NewBatchServiceAdapter(slog.Default(), nil)

	testEvent := DistributionEvent{
		Timestamp: time.Now(),
		Action:    "pull",
		Target: DistributionEventTarget{
			Repository: "my-image",
			Tag:        "v1.0",
			Digest:     "sha256:abc123",
		},
		Request: DistributionEventRequest{
			UserAgent: "docker/20.10.7 go/go1.16.4 os/linux arch/amd64",
		},
	}

	ctx := context.Background()
	start := time.Now()

	// Call SavePullEvent multiple times
	for i := 0; i < 100; i++ {
		err := adapter.SavePullEvent(ctx, &testEvent)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	}

	duration := time.Since(start)

	// Should complete very quickly (< 10ms) since it's non-blocking
	if duration > 10*time.Millisecond {
		t.Errorf("SavePullEvent took too long (%v), expected non-blocking behavior", duration)
	}
}

func TestBatchServiceAdapter_Close(t *testing.T) {
	adapter := NewBatchServiceAdapter(slog.Default(), nil)

	// Close should complete successfully
	err := adapter.Close(5 * time.Second)
	if err != nil {
		t.Errorf("Expected successful close, got error: %v", err)
	}

	// Calling Close again should be safe (idempotent)
	err = adapter.Close(5 * time.Second)
	if err != nil {
		t.Errorf("Expected Close to be idempotent, got error: %v", err)
	}
}

func TestBatchServiceAdapter_GetMetrics(t *testing.T) {
	adapter := NewBatchServiceAdapter(slog.Default(), nil)

	metrics := adapter.GetMetrics()

	// Check that all expected metrics are present
	expectedKeys := []string{
		"events_received",
		"events_dropped",
		"events_flushed",
		"batches_flushed",
		"current_queue_size",
	}

	for _, key := range expectedKeys {
		if _, exists := metrics[key]; !exists {
			t.Errorf("Expected metric %s to be present", key)
		}
	}
}

func TestBatchServiceAdapter_MetricsTracking(t *testing.T) {
	adapter := NewBatchServiceAdapter(slog.Default(), nil)

	// When pool is nil, SavePullEvent returns early without incrementing metrics
	// This test verifies the graceful degradation behavior
	testEvent := DistributionEvent{
		Timestamp: time.Now(),
		Action:    "pull",
		Target: DistributionEventTarget{
			Repository: "my-image",
			Tag:        "v1.0",
			Digest:     "sha256:abc123",
		},
		Request: DistributionEventRequest{
			UserAgent: "docker/20.10.7 go/go1.16.4 os/linux arch/amd64",
		},
	}

	ctx := context.Background()

	// Send a few events - they should be ignored when pool is nil
	for i := 0; i < 5; i++ {
		_ = adapter.SavePullEvent(ctx, &testEvent)
	}

	metrics := adapter.GetMetrics()

	// When pool is nil, events are not queued so metrics remain at 0
	if metrics["events_received"] != 0 {
		t.Errorf("Expected events_received to be 0 (pool is nil), got %d", metrics["events_received"])
	}

	// Close and verify
	_ = adapter.Close(5 * time.Second)
}

func TestGetEnvInt(t *testing.T) {
	// Test default value
	val := getEnvInt("NONEXISTENT_ENV_VAR", 42)
	if val != 42 {
		t.Errorf("Expected default value 42, got %d", val)
	}
}

func TestGetEnvDuration(t *testing.T) {
	// Test default value
	val := getEnvDuration("NONEXISTENT_ENV_VAR", 5*time.Second)
	if val != 5*time.Second {
		t.Errorf("Expected default value 5s, got %v", val)
	}
}