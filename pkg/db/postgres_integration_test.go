// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Seqera

//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seqeralabs/staticreg/pkg/db/dbtest"
)

// Run with: go test -tags=integration ./pkg/db/...
// Requires docker-compose postgres up.

func TestInitPool_CreatesDedicatedSchemaAndAppliesMigrations(t *testing.T) {
	dbtest.SetEnv(t, "staticreg_test")

	pool := InitPool()
	if pool == nil {
		t.Fatal("InitPool returned nil")
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DROP SCHEMA IF EXISTS staticreg_test CASCADE")
		pool.Close()
	})

	ctx := context.Background()

	var searchPath string
	if err := pool.QueryRow(ctx, "SHOW search_path").Scan(&searchPath); err != nil {
		t.Fatalf("show search_path: %v", err)
	}
	if searchPath != `"staticreg_test"` && searchPath != "staticreg_test" {
		t.Fatalf("expected search_path=staticreg_test, got %q", searchPath)
	}

	if !exists(t, ctx, pool, `SELECT 1 FROM information_schema.schemata WHERE schema_name = 'staticreg_test'`) {
		t.Fatal("schema staticreg_test was not created")
	}

	if !exists(t, ctx, pool, `SELECT 1 FROM information_schema.tables WHERE table_schema = 'staticreg_test' AND table_name = 'goose_db_version'`) {
		t.Fatal("goose_db_version not in staticreg_test schema")
	}

	if !exists(t, ctx, pool, `SELECT 1 FROM information_schema.tables WHERE table_schema = 'staticreg_test' AND table_name = 'container_pull_metrics'`) {
		t.Fatal("container_pull_metrics not in staticreg_test schema")
	}

	for _, idx := range []string{"idx_pull_date", "idx_repo_date", "idx_repo_arch_date"} {
		if !exists(t, ctx, pool, `SELECT 1 FROM pg_indexes WHERE schemaname = 'staticreg_test' AND indexname = $1`, idx) {
			t.Fatalf("index %s not in staticreg_test schema", idx)
		}
	}

	// Idempotent re-init: a second InitPool against the same schema must succeed.
	pool.Close()
	pool2 := InitPool()
	if pool2 == nil {
		t.Fatal("second InitPool returned nil")
	}
	defer pool2.Close()
}

func TestInitPool_RejectsInvalidSchemaName(t *testing.T) {
	dbtest.SetEnv(t, "Bad-Schema; DROP TABLE")

	if pool := InitPool(); pool != nil {
		pool.Close()
		t.Fatal("expected nil pool for invalid schema name")
	}
}

func exists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, q string, args ...any) bool {
	t.Helper()
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		t.Fatalf("query %q: %v", q, err)
	}
	defer rows.Close()
	return rows.Next()
}
