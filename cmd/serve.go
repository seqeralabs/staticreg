package cmd

import (
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serves a webserver with an HTML listing of all images and tags in a v2 registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		panic("not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
