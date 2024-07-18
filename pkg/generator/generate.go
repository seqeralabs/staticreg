package generator

import (
	"context"
	"os"
	"path"

	"github.com/regclient/regclient"
	"github.com/seqeralabs/staticreg/pkg/templates"
)

func sanitizeAbsoluteDirPath(inputPath string) string {
	cleanedPath := path.Clean(inputPath)
	if cleanedPath[len(cleanedPath)-1] != '/' {
		cleanedPath += "/"
	}
	return cleanedPath
}

func Generate(
	ctx context.Context,
	rc *regclient.RegClient,
	registryHostname string,
	baseDir string,
	absoluteDir string) error {

	absoluteDir = sanitizeAbsoluteDirPath(absoluteDir)

	repos, err := rc.RepoList(ctx, registryHostname)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	f, err := os.Create(path.Join(baseDir, "index.html"))
	if err != nil {
		return err
	}
	defer f.Close()

	baseData := templates.BaseData{
		AbsoluteDir:  absoluteDir,
		RegistryName: registryHostname,
	}

	err = templates.RenderIndex(f, templates.IndexData{
		BaseData:     baseData,
		Repositories: repos.Repositories,
	})
	if err != nil {
		return err
	}

	for _, repo := range repos.Repositories {
		repoDir := path.Join(baseDir, "repo", repo)
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			return err
		}
		f, err := os.Create(path.Join(repoDir, "index.html"))
		if err != nil {
			return err
		}
		defer f.Close()
		err = templates.RenderRepository(f, templates.RepositoryData{
			BaseData:       baseData,
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
			return err
		}
	}

	return nil

}
