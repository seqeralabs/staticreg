package generator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/seqeralabs/staticreg/pkg/templates"
)

type Generator struct {
	rc               *regclient.RegClient
	absoluteDir      string
	registryHostname string
	baseDir          string
}

func New(
	rc *regclient.RegClient,
	absoluteDir string,
	registryHostname string,
	baseDir string,
) *Generator {
	absoluteDir = sanitizeAbsoluteDirPath(absoluteDir)
	return &Generator{
		rc:               rc,
		absoluteDir:      absoluteDir,
		registryHostname: registryHostname,
		baseDir:          baseDir,
	}
}

func sanitizeAbsoluteDirPath(inputPath string) string {
	cleanPath := path.Clean(inputPath)
	if cleanPath[len(cleanPath)-1] != '/' {
		cleanPath += "/"
	}
	return cleanPath
}

func (g *Generator) Generate(
	ctx context.Context) error {
	log := logger.FromContext(ctx)

	if err := os.MkdirAll(g.baseDir, 0755); err != nil {
		return err
	}

	log.Info("generating repositories list page")
	indexFile, err := os.Create(path.Join(g.baseDir, "index.html"))
	if err != nil {
		return err
	}
	defer indexFile.Close()

	err = g.generateIndex(ctx, indexFile)
	if err != nil {
		return err
	}

	repos, err := g.rc.RepoList(ctx, g.registryHostname)
	if err != nil {
		return err
	}

	for _, repo := range repos.Repositories {
		repoLog := log.With(slog.String("repo", repo))
		repoLog.Info("generating repository page")
		repoDir := path.Join(g.baseDir, "repo", repo)
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			return err
		}
		f, err := os.Create(path.Join(repoDir, "index.html"))
		if err != nil {
			return err
		}
		defer f.Close()

		err = g.generateRepository(ctx, f, repo)
		if err != nil {
			repoLog.Error("error generating repository page, skipping...")
		}
	}

	return nil

}

func (g *Generator) tagData(ctx context.Context, repo string, tag string) (*templates.TagData, error) {
	tagFullRef := fmt.Sprintf("%s/%s:%s", g.registryHostname, repo, tag)
	tagRef, err := ref.New(tagFullRef)
	if err != nil {
		return nil, err
	}

	imageConfig, err := g.rc.ImageConfig(ctx, tagRef)
	if err != nil {
		return nil, err
	}
	innerConfig := imageConfig.GetConfig()

	return &templates.TagData{
		Name:          repo,
		Tag:           tag,
		PullReference: tagRef.CommonName(),
		CreatedAt:     innerConfig.Created.Format(time.RFC3339),
	}, nil
}

func (g *Generator) repoData(ctx context.Context, repo string) (*templates.RepositoryData, error) {
	baseData := templates.BaseData{
		AbsoluteDir:  g.absoluteDir,
		RegistryName: g.registryHostname,
	}

	log := logger.FromContext(ctx).With(slog.String("repo", repo))
	tags := []templates.TagData{}

	repoFullRef := fmt.Sprintf("%s/%s", g.registryHostname, repo)
	repoRef, err := ref.New(repoFullRef)
	if err != nil {
		return nil, err
	}

	tagList, err := g.rc.TagList(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	for _, tag := range tagList.Tags {
		tagData, err := g.tagData(ctx, repo, tag)
		if err != nil {
			log.Warn("could not generate tag data", logger.ErrAttr(err), slog.String("tag", tag))
		}
		tags = append(tags, *tagData)
	}
	repoData := &templates.RepositoryData{
		BaseData:       baseData,
		RepositoryName: repo,
		PullReference:  repoRef.CommonName(),
		Tags:           tags,
	}

	return repoData, err
}

func (g *Generator) generateRepository(
	ctx context.Context,
	w io.Writer,
	repo string,
) error {
	repoData, err := g.repoData(ctx, repo)
	if err != nil {
		return err
	}
	err = templates.RenderRepository(w, *repoData)
	if err != nil {
		return err
	}
	return nil
}

func (g *Generator) generateIndex(
	ctx context.Context,
	w io.Writer,
) error {
	log := logger.FromContext(ctx)

	repositoriesData := []templates.RepositoryData{}
	baseData := templates.BaseData{
		AbsoluteDir:  g.absoluteDir,
		RegistryName: g.registryHostname,
	}

	repos, err := g.rc.RepoList(ctx, g.registryHostname)
	if err != nil {
		return err
	}

	for _, repo := range repos.Repositories {
		repoData, err := g.repoData(ctx, repo)
		if err != nil {
			log.Warn("could not retrieve repo data", slog.String("repo", repo), logger.ErrAttr(err))
		}
		repositoriesData = append(repositoriesData, *repoData)
	}

	return templates.RenderIndex(w, templates.IndexData{
		BaseData:     baseData,
		Repositories: repositoriesData,
	})

}
