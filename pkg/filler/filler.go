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
	"fmt"
	"log/slog"
	"sort"
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
		LastUpdated:  time.Now().Format(time.RFC3339),
	}
}

func (f *Filler) RepoData(ctx context.Context, repo string) (*templates.RepositoryData, error) {
	baseData := f.BaseData()

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
