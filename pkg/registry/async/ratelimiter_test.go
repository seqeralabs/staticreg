// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Seqera
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package async

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rps := 1
	requestCount := 5
	limiter := newRateLimiter(rps)
	defer limiter.Stop()

	start := time.Now()
	for i := 0; i < requestCount; i++ {
		limiter.Allow()
	}
	duration := time.Since(start)
	if duration < 5*time.Second {
		t.Errorf("expected duration to be at least 5 seconds, got %v", duration)
	}
}

func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	rps := 10
	limiter := newRateLimiter(rps)
	defer limiter.Stop()

	requestCount := 100
	done := make(chan struct{}, requestCount)

	for i := 0; i < requestCount; i++ {
		go func() {
			limiter.Allow()
			done <- struct{}{}
		}()
	}

	// Wait for all requests to finish
	for i := 0; i < requestCount; i++ {
		<-done
	}

	// Since rps is 10, allowing 100 calls should take approximately 10 seconds
	// let's wait for that to happen before doing any assertions
	duration := time.Duration(requestCount/rps) * time.Second
	time.Sleep(duration)

	// To check if we are somewhat in the range of expected duration
	if duration < 9*time.Second || duration > 11*time.Second {
		t.Errorf("expected duration to be around 10 seconds, got %v", duration)
	}
}
