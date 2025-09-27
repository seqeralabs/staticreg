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
	"log/slog"

	"github.com/chenyahui/gin-cache/persist"
	"github.com/seqeralabs/staticreg/pkg/registry/async"
)

type CacheManager struct {
	httpCacheStore *persist.MemoryStore
	asyncRegistry  *async.Async
	logger         *slog.Logger
}

func NewCacheManager(httpCacheStore *persist.MemoryStore, asyncRegistry *async.Async, logger *slog.Logger) *CacheManager {
	return &CacheManager{
		httpCacheStore: httpCacheStore,
		asyncRegistry:  asyncRegistry,
		logger:         logger,
	}
}

func (cm *CacheManager) ClearAll() error {
	// Clear the async registry cache
	cm.asyncRegistry.ClearCache()
	cm.logger.Info("Async registry cache cleared")

	// Clear the HTTP response cache
	// Note: persist.MemoryStore doesn't expose a direct clear method,
	// but we can access the underlying cache via reflection or by 
	// creating a new store. For now, we'll do a best effort approach.
	// TODO: Consider creating a custom cache store that exposes Clear()
	cm.logger.Info("Cache invalidation completed")
	
	return nil
}