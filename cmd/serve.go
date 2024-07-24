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

		rc := registry.ClientFromConfig(*rootCfg)

		filler := filler.New(rc, rootCfg.RegistryHostname, "/")

		regServer := staticreg.New(rc, filler, rootCfg.RegistryHostname)
		// TODO: make bind addr configurable
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
