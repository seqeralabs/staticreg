// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Seqera
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package filler

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"time"

	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/seqeralabs/staticreg/pkg/registry"
	"github.com/seqeralabs/staticreg/pkg/registry/errs"
	"github.com/seqeralabs/staticreg/pkg/templates"
)

type Filler struct {
	registryHostname string
	absoluteDir      string
	regClient        registry.Client
}

func New(regClient registry.Client, registryHostname string, absoluteDir string) *Filler {
	return &Filler{
		absoluteDir:      absoluteDir,
		regClient:        regClient,
		registryHostname: registryHostname,
	}
}

func (f *Filler) TagData(ctx context.Context, repo string, tag string) (*templates.TagData, error) {
	imageInfo, reference, err := f.regClient.ImageInfo(ctx, repo, tag)
	if err != nil {
		return nil, err
	}

	cfg, err := imageInfo.ConfigFile()
	if err != nil {
		return nil, err
	}

	return &templates.TagData{
		Name:          repo,
		Tag:           tag,
		PullReference: reference,
		CreatedAt:     cfg.Created.Format(time.RFC3339),
	}, nil
}

func (f *Filler) BaseData() templates.BaseData {
	return templates.BaseData{
		AbsoluteDir:  f.absoluteDir,
		RegistryName: f.registryHostname,
		LastUpdated:  time.Now().Format(time.RFC3339),
	}
}

func (f *Filler) RepoData(ctx context.Context, repo string) (*templates.RepositoryData, error) {
	baseData := f.BaseData()

	log := logger.FromContext(ctx).With(slog.String("repo", repo))
	tags := []templates.TagData{}

	tagList, err := f.regClient.TagList(ctx, repo)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, nil
		}
	}
	if tagList == nil {
		log.Warn("tag list is nil")
		return nil, nil
	}

	for _, tag := range tagList {
		tagData, err := f.TagData(ctx, repo, tag)
		if err != nil {
			log.Warn("could not generate tag data", logger.ErrAttr(err), slog.String("tag", tag))
			continue
		}
		if tagData == nil {
			continue
		}
		tags = append(tags, *tagData)
	}
	if len(tags) == 0 {
		return nil, nil
	}

	orderedTags := orderTagsByDate(tags)
	if len(orderedTags) == 0 {
		return nil, nil
	}
	mostRecentTag := orderedTags[0]
	repoData := &templates.RepositoryData{
		BaseData:       baseData,
		RepositoryName: repo,
		PullReference:  mostRecentTag.PullReference,
		Tags:           orderedTags,
		LastUpdatedAt:  mostRecentTag.CreatedAt,
	}

	return repoData, err
}

func orderTagsByDate(tags []templates.TagData) []templates.TagData {
	sort.Slice(tags, func(i, j int) bool {
		dateI, err := time.Parse(time.RFC3339, tags[i].CreatedAt)
		if err != nil {
			return false
		}
		dateJ, err := time.Parse(time.RFC3339, tags[j].CreatedAt)
		if err != nil {
			return false
		}
		return dateI.After(dateJ)
	})
	return tags
}
