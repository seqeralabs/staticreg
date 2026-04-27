// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Seqera

//go:build integration

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/seqeralabs/staticreg/pkg/db"
	"github.com/seqeralabs/staticreg/pkg/db/dbtest"
)

// Run with: go test -tags=integration ./pkg/server/...
// Requires docker-compose postgres up.

func TestDBReadinessHandler_HappyPath(t *testing.T) {
	dbtest.SetEnv(t, "staticreg_health_test")
	pool := db.InitPool()
	if pool == nil {
		t.Fatal("InitPool returned nil")
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DROP SCHEMA IF EXISTS staticreg_health_test CASCADE")
		pool.Close()
	})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/healthz/db", dbReadinessHandler(pool))
	r.GET("/metrics/db", dbMetricsHandler(pool))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz/db", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("/healthz/db: got %d, want 200; body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics/db", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("/metrics/db: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var stats map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode metrics: %v", err)
	}
	for _, key := range []string{"acquired_conns", "idle_conns", "max_conns", "total_conns"} {
		if _, ok := stats[key]; !ok {
			t.Errorf("missing key %q in /metrics/db response: %v", key, stats)
		}
	}
}
