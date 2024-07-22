package cmd

import (
	"log/slog"

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

		log.Info("generating static website",
			slog.String("output", outputDirectory),
			slog.String("absolute-dir", absoluteDir),
		)

		gen := generator.New(rc, absoluteDir, rootCfg.RegistryHostname, outputDirectory)
		return gen.Generate(cmd.Context())

	},
}

func init() {
	generateCmd.PersistentFlags().StringVar(&outputDirectory, "output", "/tmp/generated-registry-html", "output directory")
	generateCmd.PersistentFlags().StringVar(&absoluteDir, "absolute-dir", "/tmp/generated-registry-html", "absolute URL dir, to match link base path")
	rootCmd.AddCommand(generateCmd)
}
