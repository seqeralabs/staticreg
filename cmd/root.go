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

var (
	registryHostname string
	registryUser     string
	registryPassword string
	skipTLSVerify    bool
	outputDirectory  string
	absoluteDir      string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "staticreg",
	Short: "Render an html listing of all images and tags in a v2 registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		regHost := config.Host{
			Name:     registryHostname,
			Hostname: registryHostname,
			User:     registryUser,
			Pass:     registryPassword,
			TLS:      config.TLSDisabled,
		}

		rc := regclient.New(
			regclient.WithConfigHost(regHost),
			regclient.WithDockerCerts(),
			regclient.WithDockerCreds(),
			regclient.WithUserAgent("seqera/staticreg"),
		)

		return generator.Generate(cmd.Context(), rc, regHost.Hostname, outputDirectory, absoluteDir)

	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&registryHostname, "registry", "localhost:5000", "registry hostname (default is localhost:5000)")
	rootCmd.PersistentFlags().StringVar(&registryUser, "user", "", "user (empty by default)")
	rootCmd.PersistentFlags().StringVar(&registryPassword, "password", "", "password (empty by default)")
	rootCmd.PersistentFlags().BoolVar(&skipTLSVerify, "skip-tls-verify", false, "disable TLS checks (default is false)")
	rootCmd.PersistentFlags().StringVar(&outputDirectory, "output", "/tmp/generated-registry-html", "output directory (default is /tmp/generated-registry-html)")
	rootCmd.PersistentFlags().StringVar(&absoluteDir, "absolute-dir", "/tmp/generated-registry-html", "absolute URL dir, to match link base path. (default is /tmp/generated-registry-html)")
}
