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
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/seqeralabs/staticreg/pkg/webhook"
)

// livenessHandler always returns 200. The process being able to respond to HTTP
// is the liveness signal; we deliberately do not check the database here so a
// transient DB outage does not trigger a pod restart.
func livenessHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// dbReadinessHandler returns 200 when the pool is configured and a Ping
// succeeds, 503 otherwise. Wired to a Kubernetes readiness probe so traffic is
// drained from a pod that has lost its DB connection.
func dbReadinessHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if pool == nil {
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "database pool not configured"})
			return
		}
		// Readiness probe must complete inside k8s' default 1s probe timeout.
		ctx, cancel := context.WithTimeout(c.Request.Context(), 750*time.Millisecond)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			// pgx error messages can echo connection details (host, user); keep
			// them server-side and return a generic message to the probe client.
			slog.Warn("readiness probe: database ping failed", logger.ErrAttr(err))
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "database unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// dbMetricsHandler exposes pgxpool runtime stats as JSON. The same numbers can
// be wrapped in a Prometheus collector later without changing the data source.
func dbMetricsHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if pool == nil {
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "database pool not configured"})
			return
		}
		s := pool.Stat()
		c.JSON(http.StatusOK, gin.H{
			"acquired_conns":             s.AcquiredConns(),
			"constructing_conns":         s.ConstructingConns(),
			"idle_conns":                 s.IdleConns(),
			"max_conns":                  s.MaxConns(),
			"total_conns":                s.TotalConns(),
			"new_conns_count":            s.NewConnsCount(),
			"acquire_count":              s.AcquireCount(),
			"acquire_duration_ns":        s.AcquireDuration().Nanoseconds(),
			"empty_acquire_count":        s.EmptyAcquireCount(),
			"canceled_acquire_count":     s.CanceledAcquireCount(),
			"max_lifetime_destroy_count": s.MaxLifetimeDestroyCount(),
			"max_idle_destroy_count":     s.MaxIdleDestroyCount(),
		})
	}
}

// webhookMetricsHandler exposes the batched webhook adapter's counters. Returns
// 503 with an explanation when the adapter was constructed without a DB pool
// (events are not being processed).
func webhookMetricsHandler(adapter *webhook.BatchServiceAdapter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if adapter == nil {
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "webhook adapter not configured"})
			return
		}
		c.JSON(http.StatusOK, adapter.GetMetrics())
	}
}
