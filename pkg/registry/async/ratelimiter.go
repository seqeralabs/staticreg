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
package async

import (
	"time"
)

// rateLimiter is a simple implementation of a rate limiter that
// controls the rate of function calls. It imposes a delay between
// allowed calls based on the specified requests per second (RPS).
// It utilizes time.Ticker for scheduling the allowed times.
type rateLimiter struct {
	ticker *time.Ticker
	quit   chan struct{}
}

func newRateLimiter(rps int) *rateLimiter {
	limiter := &rateLimiter{
		ticker: time.NewTicker(time.Second / time.Duration(rps)),
		quit:   make(chan struct{}),
	}
	return limiter
}

func (rl *rateLimiter) Allow() {
	<-rl.ticker.C
}

func (rl *rateLimiter) Stop() {
	close(rl.quit)
	rl.ticker.Stop()
}
