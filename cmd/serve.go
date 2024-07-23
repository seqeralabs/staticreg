package cmd

import (
	"github.com/seqeralabs/staticreg/pkg/filler"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/seqeralabs/staticreg/pkg/registry"
	"github.com/seqeralabs/staticreg/pkg/server"
	"github.com/seqeralabs/staticreg/pkg/server/staticreg"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serves a webserver with an HTML listing of all images and tags in a v2 registry",
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := cmd.Context()
		log := logger.FromContext(ctx)

		rc := registry.ClientFromConfig(*rootCfg)

		filler := filler.New(rc, rootCfg.RegistryHostname, "/")

		regServer := staticreg.New(rc, filler, rootCfg.RegistryHostname)
		// TODO: make bind addr configurable
		srv, err := server.New(":8081", regServer, log)
		if err != nil {
			return err
		}

		return srv.Start()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
