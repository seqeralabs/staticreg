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
package logger

import (
	"context"
	"io"
	"log/slog"
)

type (
	loggerKey struct{}
)

func newProduction(w io.Writer) *slog.Logger {
	level := slog.LevelInfo
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}

func newDevelopment(w io.Writer) *slog.Logger {
	level := slog.LevelDebug
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}

func New(w io.Writer, production bool) *slog.Logger {
	if production {
		return newProduction(w)
	}
	return newDevelopment(w)
}

func Context(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	logger := ctx.Value(loggerKey{})
	if logger == nil {
		logger = ctx.Value("logger")
		if logger == nil {
			return nil
		}
	}
	return logger.(*slog.Logger)
}

func ErrAttr(err error) slog.Attr {
	return slog.Any("error", err)
}
