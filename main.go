package main

import (
	"context"
	"log"
	"os"
	"path"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"

	"github.com/seqeralabs/staticreg/pkg/templates"
)

func main() {
	baseDir := "/tmp/staticgen"
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

	f, err := os.Create(path.Join(baseDir, "index.html"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatal(err)
	}

	err = templates.RenderIndex(f, templates.IndexData{
		RegistryName: regHost.Name,
		Repositories: repos.Repositories,
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, repo := range repos.Repositories {
		repoDir := path.Join(baseDir, "repo", repo)
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			log.Fatal(err)
		}
		f, err := os.Create(path.Join(repoDir, "index.html"))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		err = templates.RenderRepository(f, templates.RepositoryData{
			RegistryName:   regHost.Name,
			RepositoryName: repo,
			Tags: []templates.TagData{
				{
					Name:      repo,
					Tag:       "latest",
					CreatedAt: "date here",
				},
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	}

}
