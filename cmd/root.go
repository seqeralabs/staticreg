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
			slog.Bool("tls-enabled", rootCfg.TLSEnabled),
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
	rootCmd.PersistentFlags().StringVar(&rootCfg.RegistryHostname, "registry", "localhost:5000", "registry hostname")
	rootCmd.PersistentFlags().StringVar(&rootCfg.RegistryUser, "user", "", "user")
	rootCmd.PersistentFlags().StringVar(&rootCfg.RegistryPassword, "password", "", "password")
	rootCmd.PersistentFlags().BoolVar(&rootCfg.SkipTLSVerify, "skip-tls-verify", false, "disable TLS certificate checks")
	rootCmd.PersistentFlags().BoolVar(&rootCfg.TLSEnabled, "tls-enable", false, "enable TLS")
	rootCmd.PersistentFlags().BoolVar(&rootCfg.LogInJSON, "json-logging", false, "log in JSON")
}
