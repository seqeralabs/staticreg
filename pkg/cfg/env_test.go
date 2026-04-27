// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Seqera

package cfg

import (
	"testing"
	"time"
)

func TestEnvInt(t *testing.T) {
	t.Run("unset returns default", func(t *testing.T) {
		if got := EnvInt("STATICREG_TEST_NONEXISTENT", 42); got != 42 {
			t.Errorf("got %d, want 42", got)
		}
	})

	t.Run("parses valid int", func(t *testing.T) {
		t.Setenv("STATICREG_TEST_INT", "7")
		if got := EnvInt("STATICREG_TEST_INT", 42); got != 7 {
			t.Errorf("got %d, want 7", got)
		}
	})

	t.Run("invalid value falls back to default", func(t *testing.T) {
		t.Setenv("STATICREG_TEST_INT", "not-a-number")
		if got := EnvInt("STATICREG_TEST_INT", 42); got != 42 {
			t.Errorf("got %d, want 42", got)
		}
	})
}

func TestEnvDuration(t *testing.T) {
	t.Run("unset returns default", func(t *testing.T) {
		if got := EnvDuration("STATICREG_TEST_NONEXISTENT", 5*time.Second); got != 5*time.Second {
			t.Errorf("got %v, want 5s", got)
		}
	})

	t.Run("parses valid duration", func(t *testing.T) {
		t.Setenv("STATICREG_TEST_DUR", "30s")
		if got := EnvDuration("STATICREG_TEST_DUR", 5*time.Second); got != 30*time.Second {
			t.Errorf("got %v, want 30s", got)
		}
	})

	t.Run("invalid value falls back to default", func(t *testing.T) {
		t.Setenv("STATICREG_TEST_DUR", "five seconds")
		if got := EnvDuration("STATICREG_TEST_DUR", 5*time.Second); got != 5*time.Second {
			t.Errorf("got %v, want 5s", got)
		}
	})
}
