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
package registry

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/seqeralabs/staticreg/pkg/cfg"
)

const defaultUserAgent = "seqera/staticreg"

var (
	uaOption = remote.WithUserAgent(defaultUserAgent)
)

type config struct {
	Registry      string
	User          string
	Password      string
	SkipTLSVerify bool
	TLSEnabled    bool
}

type Registry struct {
	cfg config
}

func (c *Registry) RepoName(r string) (name.Repository, error) {
	return name.NewRepository(r, name.WithDefaultRegistry(c.cfg.Registry))
}

func (c *Registry) RepoList(ctx context.Context) ([]string, error) {
	reg, _ := name.NewRegistry(c.cfg.Registry)
	repos, err := remote.Catalog(ctx, reg)
	if err != nil {
		return nil, err
	}
	reposret := []string{}
	for _, r := range repos {
		repoName, err := c.RepoName(r)
		if err != nil {
			return nil, err
		}
		reposret = append(reposret, repoName.RepositoryStr())
	}
	return reposret, nil
}

func (c *Registry) TagList(ctx context.Context, repo string) ([]string, error) {
	rname, err := name.NewRepository(repo, name.WithDefaultRegistry(c.cfg.Registry))
	if err != nil {
		return nil, err
	}
	return remote.List(rname, remote.WithContext(ctx), uaOption)
}

func (c *Registry) ImageInfo(ctx context.Context, image string, tag string) (v1.Image, string, error) {
	ref, err := name.ParseReference(fmt.Sprintf("%s/%s:%s", c.cfg.Registry, image, tag))
	if err != nil {
		return nil, "", err
	}
	i, err := remote.Image(ref, remote.WithContext(ctx), uaOption)
	if err != nil {
		return nil, "", err
	}

	return i, ref.String(), nil
}

func New(rootCfg *cfg.Root) *Registry {
	cfg := config{
		Registry:      rootCfg.RegistryHostname,
		User:          rootCfg.RegistryUser,
		Password:      rootCfg.RegistryPassword,
		TLSEnabled:    rootCfg.TLSEnabled,
		SkipTLSVerify: rootCfg.SkipTLSVerify,
	}

	return &Registry{
		cfg: cfg,
	}
}
