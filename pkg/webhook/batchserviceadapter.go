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
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seqeralabs/staticreg/pkg/cfg"
)

// BatchServiceAdapter processes webhook events asynchronously with batching
// to reduce database load and improve webhook response times
type BatchServiceAdapter struct {
	Logger        *slog.Logger
	Pool          *pgxpool.Pool
	eventChan     chan *DistributionEvent
	batchSize     int
	flushInterval time.Duration
	bufferSize    int
	wg            sync.WaitGroup
	stopOnce      sync.Once
	metrics       struct {
		eventsReceived   atomic.Int64
		eventsDropped    atomic.Int64
		eventsFlushed    atomic.Int64
		batchesFlushed   atomic.Int64
		currentQueueSize atomic.Int64
	}
}

// NewBatchServiceAdapter creates a new batch webhook service adapter
// Configuration is pulled from environment variables with sensible defaults:
//   - STATICREG_METRICS_BATCH_SIZE: Number of events to batch before flushing (default: 100)
//   - STATICREG_METRICS_FLUSH_INTERVAL: Time interval to force flush (default: 5s)
//   - STATICREG_METRICS_BUFFER_SIZE: Channel buffer size (default: 10000)
func NewBatchServiceAdapter(log *slog.Logger, pool *pgxpool.Pool) *BatchServiceAdapter {
	batchSize := cfg.EnvInt("STATICREG_METRICS_BATCH_SIZE", 100)
	flushInterval := cfg.EnvDuration("STATICREG_METRICS_FLUSH_INTERVAL", 5*time.Second)
	bufferSize := cfg.EnvInt("STATICREG_METRICS_BUFFER_SIZE", 10000)

	adapter := &BatchServiceAdapter{
		Logger:        log,
		Pool:          pool,
		eventChan:     make(chan *DistributionEvent, bufferSize),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		bufferSize:    bufferSize,
	}

	if pool != nil {
		adapter.wg.Add(1)
		go adapter.worker()
		log.Info("Started batched webhook processor",
			"batchSize", batchSize,
			"flushInterval", flushInterval,
			"bufferSize", bufferSize)
	} else {
		log.Debug("Database pool is nil, batched webhook processor not started")
	}

	return adapter
}

// SavePullEvent queues an event for asynchronous processing
// Returns immediately without blocking on database operations
func (a *BatchServiceAdapter) SavePullEvent(ctx context.Context, event *DistributionEvent) error {
	if a.Pool == nil {
		a.Logger.Debug("Skipping pull event save: Database pool is not active.")
		return nil
	}

	a.metrics.eventsReceived.Add(1)

	select {
	case a.eventChan <- event:
		a.metrics.currentQueueSize.Add(1)
	default:
		// Buffer is full - drop event and log warning
		a.metrics.eventsDropped.Add(1)
		a.Logger.Warn("Event buffer full, dropping event",
			"repository", event.Target.Repository,
			"tag", event.Target.Tag,
			"digest", event.Target.Digest,
			"dropped_total", a.metrics.eventsDropped.Load(),
			"queue_size", a.bufferSize)
	}

	return nil
}

// Close gracefully shuts down the worker, flushing any pending events
func (a *BatchServiceAdapter) Close(timeout time.Duration) error {
	var closeErr error
	a.stopOnce.Do(func() {
		if a.Pool == nil {
			return
		}

		a.Logger.Info("Shutting down batched webhook processor",
			"timeout", timeout,
			"pending_events", a.metrics.currentQueueSize.Load())

		close(a.eventChan)

		done := make(chan struct{})
		go func() {
			a.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			a.Logger.Info("Batched webhook processor shutdown complete",
				"events_received", a.metrics.eventsReceived.Load(),
				"events_flushed", a.metrics.eventsFlushed.Load(),
				"events_dropped", a.metrics.eventsDropped.Load(),
				"batches_flushed", a.metrics.batchesFlushed.Load())
		case <-time.After(timeout):
			closeErr = fmt.Errorf("shutdown timeout after %v, some events may be lost", timeout)
			a.Logger.Error("Failed to gracefully shutdown webhook service",
				"error", closeErr,
				"pending_events", a.metrics.currentQueueSize.Load())
		}
	})
	return closeErr
}

// worker processes events in batches
func (a *BatchServiceAdapter) worker() {
	defer a.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			a.Logger.Error("Worker panicked, attempting recovery", "panic", r)
			// Restart worker if panic occurs
			a.wg.Add(1)
			go a.worker()
		}
	}()

	ticker := time.NewTicker(a.flushInterval)
	defer ticker.Stop()

	batch := make([]*DistributionEvent, 0, a.batchSize)

	for {
		select {
		case event, ok := <-a.eventChan:
			if !ok {
				// Channel closed, flush remaining events and exit
				if len(batch) > 0 {
					a.flush(batch)
				}
				return
			}

			// If this is the first event in a new batch, reset the ticker
			if len(batch) == 0 {
				ticker.Reset(a.flushInterval)
			}

			batch = append(batch, event)
			a.metrics.currentQueueSize.Add(-1)

			if len(batch) >= a.batchSize {
				a.flush(batch)
				batch = batch[:0] // Clear batch, reuse capacity
			}

		case <-ticker.C:
			if len(batch) > 0 {
				a.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

// flush writes a batch of events to the database using pgx.Batch for efficiency
func (a *BatchServiceAdapter) flush(batch []*DistributionEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()

	// Use pgx.Batch for efficient bulk inserts with pipelining
	pgxBatch := &pgx.Batch{}

	query := `
        INSERT INTO container_pull_metrics (pull_date, repo_name, tag, digest, architecture, pull_count, updated_at)
        VALUES ($1, $2, $3, $4, $5, 1, NOW())
        ON CONFLICT (pull_date, repo_name, tag, digest, architecture)
        DO UPDATE SET
            pull_count = container_pull_metrics.pull_count + 1,
            updated_at = NOW()`

	for _, event := range batch {
		pgxBatch.Queue(query,
			event.Timestamp.UTC().Format("2006-01-02"),
			event.Target.Repository,
			event.Target.Tag,
			event.Target.Digest,
			event.GetArchitecture(),
		)
	}

	results := a.Pool.SendBatch(ctx, pgxBatch)
	defer results.Close()

	errorCount := 0
	for i := 0; i < len(batch); i++ {
		if _, err := results.Exec(); err != nil {
			errorCount++
			if errorCount <= 3 { // Log first few errors only
				a.Logger.Error("Failed to execute batch insert",
					"error", err,
					"event_index", i,
					"repository", batch[i].Target.Repository)
			}
		}
	}

	successCount := len(batch) - errorCount
	a.metrics.eventsFlushed.Add(int64(successCount))
	a.metrics.batchesFlushed.Add(1)

	duration := time.Since(startTime)
	a.Logger.Info("Flushed event batch",
		"batch_size", len(batch),
		"success_count", successCount,
		"error_count", errorCount,
		"duration_ms", duration.Milliseconds(),
		"total_flushed", a.metrics.eventsFlushed.Load(),
		"total_batches", a.metrics.batchesFlushed.Load())

	if errorCount > 0 {
		a.Logger.Warn("Batch flush completed with errors",
			"error_count", errorCount,
			"batch_size", len(batch))
	}
}

// GetMetrics returns current metrics for observability
func (a *BatchServiceAdapter) GetMetrics() map[string]int64 {
	return map[string]int64{
		"events_received":    a.metrics.eventsReceived.Load(),
		"events_dropped":     a.metrics.eventsDropped.Load(),
		"events_flushed":     a.metrics.eventsFlushed.Load(),
		"batches_flushed":    a.metrics.batchesFlushed.Load(),
		"current_queue_size": a.metrics.currentQueueSize.Load(),
	}
}

