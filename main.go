package main

import (
	"context"
	"log"
	"os"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"

	"github.com/seqeralabs/staticreg/pkg/templates"
)

func main() {
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

	ctx := context.Background()
	repos, err := rc.RepoList(ctx, regHost.Hostname)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create("/tmp/index.html")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	err = templates.RenderIndex(f, templates.IndexData{
		RegistryName: regHost.Name,
		Repositories: repos.Repositories,
	})
	if err != nil {
		log.Fatal(err)
	}

}
