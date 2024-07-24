package cmd

import (
	"log/slog"
	"os"

	"github.com/seqeralabs/staticreg/pkg/cfg"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/spf13/cobra"
)

var rootCfg *cfg.Root = &cfg.Root{}

var rootCmd = &cobra.Command{
	Use:   "staticreg",
	Short: "A tool to browse images in an OCI registry",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		log := logger.New(cmd.OutOrStderr(), rootCfg.LogInJSON)
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
}
