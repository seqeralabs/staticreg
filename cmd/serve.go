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
package cmd

import (
	"time"

	"log/slog"

	"github.com/seqeralabs/staticreg/pkg/filler"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/seqeralabs/staticreg/pkg/registry/async"
	"github.com/seqeralabs/staticreg/pkg/registry/registry"
	"github.com/seqeralabs/staticreg/pkg/server"
	"github.com/seqeralabs/staticreg/pkg/server/staticreg"
	"github.com/spf13/cobra"
)

var (
	bindAddr          string
	ignoredUserAgents []string
	cacheDuration     time.Duration
	refreshInterval   time.Duration
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serves a webserver with an HTML listing of all images and tags in a v2 registry",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		log := logger.FromContext(ctx)
		log.Info("starting server",
			slog.Duration("cache-duration", cacheDuration),
			slog.String("bind-addr", bindAddr),
			slog.Any("ignored-user-agents", ignoredUserAgents),
			slog.Any("refresh-interval", refreshInterval),
		)

		client := registry.New(rootCfg)
		asyncClient := async.New(client, refreshInterval)

		filler := filler.New(asyncClient, rootCfg.RegistryHostname, "/")

		regServer := staticreg.New(asyncClient, filler, rootCfg.RegistryHostname)
		srv, err := server.New(bindAddr, regServer, log, cacheDuration, ignoredUserAgents)
		if err != nil {
			slog.Error("error creating server", logger.ErrAttr(err))
			return
		}

		errCh := make(chan error, 1)
		go func() {
			errCh <- srv.Start()
		}()

		go func() {
			errCh <- asyncClient.Start(ctx)
		}()

		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			if err == nil {
				slog.Error("operations exited unexpectedly")
				return
			}
			slog.Error("unexpected error", logger.ErrAttr(err))
			return
		}
	},
}

func init() {
	serveCmd.PersistentFlags().StringVar(&bindAddr, "bind-addr", "127.0.0.1:8093", "server bind address")
	serveCmd.PersistentFlags().StringArrayVar(&ignoredUserAgents, "ignored-user-agent", []string{}, "user agents to ignore (reply with empty body and 200 OK). A user agent is ignored if it contains the one of the values passed to this flag")
	serveCmd.PersistentFlags().DurationVar(&cacheDuration, "cache-duration", time.Minute*1, "how long to keep a generated page in cache before expiring it, 0 to never expire")
	serveCmd.PersistentFlags().DurationVar(&cacheDuration, "refresh-interval", time.Minute*15, "how long to wait before trying to get fresh data from the target registry")
	rootCmd.AddCommand(serveCmd)
}
