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
	"path"

	"github.com/seqeralabs/staticreg/pkg/filler"
	"github.com/seqeralabs/staticreg/pkg/generator"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/seqeralabs/staticreg/pkg/registry"
	"github.com/spf13/cobra"
)

var (
	outputDirectory string
	absoluteDir     string
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Render an html listing of all images and tags in a v2 registry to an output directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		log := logger.FromContext(ctx)

		rc := registry.ClientFromConfig(*rootCfg)

		sanitizedAbsoluteDir := sanitizeAbsoluteDirPath(absoluteDir)
		log.Info("generating static website",
			slog.String("output", outputDirectory),
			slog.String("absolute-dir", sanitizedAbsoluteDir),
		)

		filler := filler.New(rc, rootCfg.RegistryHostname, sanitizedAbsoluteDir)
		gen := generator.New(rc, filler, sanitizedAbsoluteDir, rootCfg.RegistryHostname, outputDirectory)
		return gen.Generate(cmd.Context())

	},
}

func init() {
	generateCmd.PersistentFlags().StringVar(&outputDirectory, "output", "/tmp/generated-registry-html", "output directory")
	generateCmd.PersistentFlags().StringVar(&absoluteDir, "absolute-dir", "/tmp/generated-registry-html", "absolute URL dir, to match link base path")
	rootCmd.AddCommand(generateCmd)
}

func sanitizeAbsoluteDirPath(inputPath string) string {
	cleanPath := path.Clean(inputPath)
	if cleanPath[len(cleanPath)-1] != '/' {
		cleanPath += "/"
	}
	return cleanPath
}
