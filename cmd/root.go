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
	"log/slog"
	"os"

	"github.com/seqeralabs/staticreg/pkg/cfg"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/spf13/cobra"

	_ "github.com/breml/rootcerts"
)

var rootCfg *cfg.Root = &cfg.Root{}

var rootCmd = &cobra.Command{
	Use:   "staticreg",
	Short: "A tool to browse images in an OCI registry",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		log := logger.New(cmd.OutOrStderr(), rootCfg.LogInJSON, rootCfg.Verbose)

		ctx = logger.Context(ctx, log)
		cmd.SetContext(ctx)

		log.Info(
			"staticreg running with options",
			slog.String("registry", rootCfg.RegistryHostname),
			slog.Bool("skip-tls-verify", rootCfg.SkipTLSVerify),
			slog.Bool("tls-enable", rootCfg.TLSEnabled),
			slog.String("user", rootCfg.RegistryUser),
			slog.String("password", func() string {
				if len(rootCfg.RegistryPassword) > 0 {
					return "[redacted]"
				}
				return "[not provided]"
			}()),
		)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	defaultRegistry := "localhost:5000"
	envRegistry := os.Getenv("REGISTRY_HOSTNAME")
	if len(envRegistry) > 0 {
		defaultRegistry = envRegistry
	}
	rootCmd.PersistentFlags().StringVar(&rootCfg.RegistryHostname, "registry", defaultRegistry, "registry hostname, can be set via the env var REGISTRY_HOSTNAME as well")
	rootCmd.PersistentFlags().StringVar(&rootCfg.RegistryUser, "user", os.Getenv("REGISTRY_USER"), "registry user to use for authentication against the provided registry, can be set via the env var REGISTRY_USER as well")
	rootCmd.PersistentFlags().StringVar(&rootCfg.RegistryPassword, "password", os.Getenv("REGISTRY_PASSWORD"), "registry password to use for authentication against the provided registry, can be set via the env var REGISTRY_PASSWORD as well")
	rootCmd.PersistentFlags().BoolVar(&rootCfg.SkipTLSVerify, "skip-tls-verify", false, "disable TLS certificate checks")
	rootCmd.PersistentFlags().BoolVar(&rootCfg.TLSEnabled, "tls-enable", false, "enable TLS")
	rootCmd.PersistentFlags().BoolVar(&rootCfg.LogInJSON, "json-logging", false, "log in JSON")
	rootCmd.PersistentFlags().BoolVar(&rootCfg.Verbose, "verbose", false, "enable verbose logging")
}
