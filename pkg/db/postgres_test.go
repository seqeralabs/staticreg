// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Seqera

package db

import (
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func clearDBEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"STATICREG_DB_URL",
		"STATICREG_DB_HOST",
		"STATICREG_DB_PORT",
		"STATICREG_DB_USER",
		"STATICREG_DB_PASSWORD",
		"STATICREG_DB_NAME",
		"STATICREG_DB_SSLMODE",
		"STATICREG_DB_SCHEMA",
		"STATICREG_DB_MAX_CONNS",
		"STATICREG_DB_MIN_CONNS",
		"STATICREG_DB_MAX_CONN_LIFETIME",
		"STATICREG_DB_MAX_CONN_IDLE_TIME",
		"STATICREG_DB_HEALTHCHECK_PERIOD",
	} {
		t.Setenv(k, "")
	}
}

func TestBuildConnectionString(t *testing.T) {
	t.Run("URL is returned verbatim", func(t *testing.T) {
		clearDBEnv(t)
		url := "postgres://u:p@h:5432/db?sslmode=require"
		t.Setenv("STATICREG_DB_URL", url)
		if got := buildConnectionString(); got != url {
			t.Fatalf("got %q, want %q", got, url)
		}
	})

	t.Run("missing required fields returns empty", func(t *testing.T) {
		clearDBEnv(t)
		t.Setenv("STATICREG_DB_HOST", "h")
		// USER and NAME unset
		if got := buildConnectionString(); got != "" {
			t.Fatalf("got %q, want empty", got)
		}
	})

	t.Run("sslmode defaults to require when unset", func(t *testing.T) {
		clearDBEnv(t)
		t.Setenv("STATICREG_DB_HOST", "h")
		t.Setenv("STATICREG_DB_USER", "u")
		t.Setenv("STATICREG_DB_NAME", "db")
		got := buildConnectionString()
		if !strings.Contains(got, "sslmode=require") {
			t.Fatalf("expected sslmode=require in %q", got)
		}
	})

	t.Run("explicit sslmode is respected", func(t *testing.T) {
		clearDBEnv(t)
		t.Setenv("STATICREG_DB_HOST", "h")
		t.Setenv("STATICREG_DB_USER", "u")
		t.Setenv("STATICREG_DB_NAME", "db")
		t.Setenv("STATICREG_DB_SSLMODE", "verify-full")
		got := buildConnectionString()
		if !strings.Contains(got, "sslmode=verify-full") {
			t.Fatalf("expected sslmode=verify-full in %q", got)
		}
		if strings.Contains(got, "sslmode=require") {
			t.Fatalf("did not expect sslmode=require in %q", got)
		}
	})

	t.Run("explicit sslmode=disable is respected", func(t *testing.T) {
		clearDBEnv(t)
		t.Setenv("STATICREG_DB_HOST", "h")
		t.Setenv("STATICREG_DB_USER", "u")
		t.Setenv("STATICREG_DB_NAME", "db")
		t.Setenv("STATICREG_DB_SSLMODE", "disable")
		got := buildConnectionString()
		if !strings.Contains(got, "sslmode=disable") {
			t.Fatalf("expected sslmode=disable in %q", got)
		}
	})

	t.Run("optional fields are appended when set", func(t *testing.T) {
		clearDBEnv(t)
		t.Setenv("STATICREG_DB_HOST", "h")
		t.Setenv("STATICREG_DB_USER", "u")
		t.Setenv("STATICREG_DB_NAME", "db")
		t.Setenv("STATICREG_DB_PORT", "5433")
		t.Setenv("STATICREG_DB_PASSWORD", "secret")
		got := buildConnectionString()
		for _, want := range []string{"port=5433", "password=secret"} {
			if !strings.Contains(got, want) {
				t.Fatalf("expected %q in %q", want, got)
			}
		}
	})
}

func TestApplyPoolTuning(t *testing.T) {
	t.Run("defaults applied when env unset", func(t *testing.T) {
		clearDBEnv(t)
		c, err := pgxpool.ParseConfig("postgres://u:p@h/db")
		if err != nil {
			t.Fatal(err)
		}
		applyPoolTuning(c)
		if c.MaxConns != 25 {
			t.Errorf("MaxConns: got %d, want 25", c.MaxConns)
		}
		if c.MinConns != 2 {
			t.Errorf("MinConns: got %d, want 2", c.MinConns)
		}
		if c.MaxConnLifetime != time.Hour {
			t.Errorf("MaxConnLifetime: got %v, want 1h", c.MaxConnLifetime)
		}
		if c.MaxConnIdleTime != 30*time.Minute {
			t.Errorf("MaxConnIdleTime: got %v, want 30m", c.MaxConnIdleTime)
		}
		if c.HealthCheckPeriod != time.Minute {
			t.Errorf("HealthCheckPeriod: got %v, want 1m", c.HealthCheckPeriod)
		}
	})

	t.Run("env overrides apply", func(t *testing.T) {
		clearDBEnv(t)
		t.Setenv("STATICREG_DB_MAX_CONNS", "100")
		t.Setenv("STATICREG_DB_MIN_CONNS", "10")
		t.Setenv("STATICREG_DB_MAX_CONN_LIFETIME", "2h")
		t.Setenv("STATICREG_DB_MAX_CONN_IDLE_TIME", "15m")
		t.Setenv("STATICREG_DB_HEALTHCHECK_PERIOD", "30s")

		c, err := pgxpool.ParseConfig("postgres://u:p@h/db")
		if err != nil {
			t.Fatal(err)
		}
		applyPoolTuning(c)
		if c.MaxConns != 100 {
			t.Errorf("MaxConns: got %d, want 100", c.MaxConns)
		}
		if c.MinConns != 10 {
			t.Errorf("MinConns: got %d, want 10", c.MinConns)
		}
		if c.MaxConnLifetime != 2*time.Hour {
			t.Errorf("MaxConnLifetime: got %v, want 2h", c.MaxConnLifetime)
		}
		if c.MaxConnIdleTime != 15*time.Minute {
			t.Errorf("MaxConnIdleTime: got %v, want 15m", c.MaxConnIdleTime)
		}
		if c.HealthCheckPeriod != 30*time.Second {
			t.Errorf("HealthCheckPeriod: got %v, want 30s", c.HealthCheckPeriod)
		}
	})
}
