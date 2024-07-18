/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/seqeralabs/staticreg/pkg/generator"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "staticreg",
	Short: "Render an html listing of all images and tags in a v2 registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		regHost := config.Host{
			Name:     "localhost:5000",
			Hostname: "localhost:5000",
			// User:     "",
			// Pass:     "",
			TLS: config.TLSDisabled,
		}

		rc := regclient.New(
			regclient.WithConfigHost(regHost),
			regclient.WithDockerCerts(),
			regclient.WithDockerCreds(),
			regclient.WithUserAgent("seqera/staticreg"),
		)

		return generator.Generate(cmd.Context(), rc, regHost.Hostname, "/tmp/generated-html", "/tmp/generated-html")

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.staticreg.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
