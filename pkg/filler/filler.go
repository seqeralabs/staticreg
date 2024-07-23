package filler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/errs"
	"github.com/regclient/regclient/types/ref"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/seqeralabs/staticreg/pkg/templates"
)

type Filler struct {
	registryHostname string
	absoluteDir      string
	rc               *regclient.RegClient
}

func New(rc *regclient.RegClient, registryHostname string, absoluteDir string) *Filler {
	return &Filler{
		absoluteDir:      absoluteDir,
		rc:               rc,
		registryHostname: registryHostname,
	}
}

func (f *Filler) TagData(ctx context.Context, repo string, tag string) (*templates.TagData, error) {
	tagFullRef := fmt.Sprintf("%s/%s:%s", f.registryHostname, repo, tag)
	tagRef, err := ref.New(tagFullRef)
	if err != nil {
		return nil, err
	}

	imageConfig, err := f.rc.ImageConfig(ctx, tagRef)
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

func (f *Filler) BaseData() templates.BaseData {
	return templates.BaseData{
		AbsoluteDir:  f.absoluteDir,
		RegistryName: f.registryHostname,
	}
}

func (f *Filler) RepoData(ctx context.Context, repo string) (*templates.RepositoryData, error) {
	baseData := templates.BaseData{
		AbsoluteDir:  f.absoluteDir,
		RegistryName: f.registryHostname,
	}

	log := logger.FromContext(ctx).With(slog.String("repo", repo))
	tags := []templates.TagData{}

	repoFullRef := fmt.Sprintf("%s/%s", f.registryHostname, repo)
	repoRef, err := ref.New(repoFullRef)
	if err != nil {
		return nil, err
	}

	tagList, err := f.rc.TagList(ctx, repoRef)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, nil
		}
	}

	for _, tag := range tagList.Tags {
		tagData, err := f.TagData(ctx, repo, tag)
		if err != nil {
			log.Warn("could not generate tag data", logger.ErrAttr(err), slog.String("tag", tag))
			continue
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
