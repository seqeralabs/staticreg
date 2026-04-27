// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Seqera

package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/seqeralabs/staticreg/pkg/webhook"
)

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestLivenessHandler_AlwaysOK(t *testing.T) {
	r := newTestRouter(t)
	r.GET("/healthz", livenessHandler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body: got %v, want status=ok", body)
	}
}

func TestDBReadinessHandler_NilPoolReturns503(t *testing.T) {
	r := newTestRouter(t)
	r.GET("/healthz/db", dbReadinessHandler(nil))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz/db", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d, want 503", w.Code)
	}
}

func TestDBMetricsHandler_NilPoolReturns503(t *testing.T) {
	r := newTestRouter(t)
	r.GET("/metrics/db", dbMetricsHandler(nil))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics/db", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d, want 503", w.Code)
	}
}

func TestWebhookMetricsHandler_NilAdapterReturns503(t *testing.T) {
	r := newTestRouter(t)
	r.GET("/metrics/webhook", webhookMetricsHandler(nil))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics/webhook", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d, want 503", w.Code)
	}
}

func TestWebhookMetricsHandler_AdapterWithoutPoolReturnsZeroes(t *testing.T) {
	// An adapter built with a nil pool still constructs successfully — no
	// worker is started, but GetMetrics() should return the zeroed counters.
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	adapter := webhook.NewBatchServiceAdapter(log, nil)

	r := newTestRouter(t)
	r.GET("/metrics/webhook", webhookMetricsHandler(adapter))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics/webhook", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	var body map[string]int64
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, key := range []string{"events_received", "events_dropped", "events_flushed", "batches_flushed", "current_queue_size"} {
		if _, ok := body[key]; !ok {
			t.Errorf("missing key %q in response", key)
		}
	}
}
