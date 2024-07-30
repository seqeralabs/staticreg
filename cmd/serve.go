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
	"github.com/seqeralabs/staticreg/pkg/registry"
	"github.com/seqeralabs/staticreg/pkg/server"
	"github.com/seqeralabs/staticreg/pkg/server/staticreg"
	"github.com/spf13/cobra"
)

var (
	bindAddr      string
	cacheDuration time.Duration
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serves a webserver with an HTML listing of all images and tags in a v2 registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		log := logger.FromContext(ctx)
		log.Info("starting server",
			slog.Duration("cache-duration", cacheDuration),
			slog.String("bind-addr", bindAddr),
		)

		client := registry.New(rootCfg)

		filler := filler.New(client, rootCfg.RegistryHostname, "/")

		regServer := staticreg.New(client, filler, rootCfg.RegistryHostname)
		srv, err := server.New(bindAddr, regServer, log, cacheDuration)
		if err != nil {
			return err
		}

		return srv.Start()
	},
}

func init() {
	serveCmd.PersistentFlags().StringVar(&bindAddr, "bind-addr", "127.0.0.1:8093", "server bind address")
	serveCmd.PersistentFlags().DurationVar(&cacheDuration, "cache-duration", time.Minute*10, "how long to keep a generated page in cache before expiring it, 0 to never expire")
	rootCmd.AddCommand(serveCmd)
}
