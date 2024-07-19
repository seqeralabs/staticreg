package generator

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
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

	log := logger.FromContext(ctx)

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

	repositoriesData := []templates.RepositoryData{}

	for _, repo := range repos.Repositories {
		repoFullRef := fmt.Sprintf("%s/%s", registryHostname, repo)
		repoRef, err := ref.New(repoFullRef)

		if err != nil {
			log.Warn("could not create repo ref", logger.ErrAttr(err))
			continue
		}

		repoDir := path.Join(baseDir, "repo", repo)
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			return err
		}
		f, err := os.Create(path.Join(repoDir, "index.html"))
		if err != nil {
			return err
		}
		defer f.Close()
		tags := []templates.TagData{}

		tagList, err := rc.TagList(ctx, repoRef)
		if err != nil {
			log.Warn("unable to create tag list for repo", logger.ErrAttr(err))
			continue
		}

		for _, tag := range tagList.Tags {
			tagFullRef := fmt.Sprintf("%s/%s:%s", registryHostname, repo, tag)
			tagRef, err := ref.New(tagFullRef)
			if err != nil {
				continue
			}

			imageConfig, err := rc.ImageConfig(ctx, tagRef)
			if err != nil {
				continue
			}
			innerConfig := imageConfig.GetConfig()

			tags = append(tags, templates.TagData{
				Name:          repo,
				Tag:           tag,
				PullReference: tagRef.CommonName(),
				CreatedAt:     innerConfig.Created.Format(time.RFC3339),
			})
		}

		repoData := templates.RepositoryData{
			BaseData:       baseData,
			RepositoryName: repo,
			PullReference:  repoRef.CommonName(),
			Tags:           tags,
		}
		repositoriesData = append(repositoriesData, repoData)

		err = templates.RenderRepository(f, repoData)
		if err != nil {
			return err
		}

	}

	err = templates.RenderIndex(f, templates.IndexData{
		BaseData:     baseData,
		Repositories: repositoriesData,
	})
	if err != nil {
		return err
	}

	return nil

}
