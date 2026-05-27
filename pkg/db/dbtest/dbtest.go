// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Seqera

//go:build integration

// Package dbtest provides integration-test helpers shared across packages
// that exercise a real postgres connection. Compiled only with the
// `integration` build tag so it does not enter normal builds.
package dbtest

import "testing"

// SetEnv populates the STATICREG_DB_* environment variables to point at the
// docker-compose postgres container with the given dedicated schema. All
// values are restored automatically by t.Setenv on cleanup.
func SetEnv(t *testing.T, schema string) {
	t.Helper()
	t.Setenv("STATICREG_DB_HOST", "127.0.0.1")
	t.Setenv("STATICREG_DB_PORT", "5432")
	t.Setenv("STATICREG_DB_USER", "staticreg")
	t.Setenv("STATICREG_DB_PASSWORD", "password")
	t.Setenv("STATICREG_DB_NAME", "staticreg")
	t.Setenv("STATICREG_DB_SSLMODE", "disable")
	t.Setenv("STATICREG_DB_SCHEMA", schema)
}
