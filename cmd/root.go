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
		log := logger.New(cmd.OutOrStderr(), true)
		ctx = logger.Context(ctx, log)
		cmd.SetContext(ctx)

		log.Info(
			"staticreg running with options",
			slog.String("registry", rootCfg.RegistryHostname),
			slog.Bool("skip-tls-verify", rootCfg.SkipTLSVerify),
			slog.Bool("tls-disabled", rootCfg.TLSDisabled),
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
	rootCmd.PersistentFlags().StringVar(&rootCfg.RegistryHostname, "registry", "localhost:5000", "registry hostname (default is localhost:5000)")
	rootCmd.PersistentFlags().StringVar(&rootCfg.RegistryUser, "user", "", "user (empty by default)")
	rootCmd.PersistentFlags().StringVar(&rootCfg.RegistryPassword, "password", "", "password (empty by default)")
	rootCmd.PersistentFlags().BoolVar(&rootCfg.SkipTLSVerify, "skip-tls-verify", false, "disable TLS certificate checks (default is false)")
	rootCmd.PersistentFlags().BoolVar(&rootCfg.TLSDisabled, "tls-disabled", true, "disable TLS (default is false)")
}
