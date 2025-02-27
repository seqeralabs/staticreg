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
	"embed"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/seqeralabs/staticreg/pkg/cfg"
	"github.com/seqeralabs/staticreg/pkg/registry"
	"html/template"
	"strings"
)

//go:embed img/*
var icons embed.FS

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
	WaveServerUrl string
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

func (c *Registry) ImageInfo(ctx context.Context, image string, tag string) (registry.ImageInfo, error) {
	ref, err := name.ParseReference(fmt.Sprintf("%s/%s:%s", c.cfg.Registry, image, tag))
	if err != nil {
		return registry.ImageInfo{}, err
	}
	i, err := remote.Image(ref, remote.WithContext(ctx), uaOption)
	if err != nil {
		return registry.ImageInfo{}, err
	}

	waveServerUrl := c.cfg.WaveServerUrl
	index, err := remote.Index(ref, remote.WithContext(ctx), uaOption)
	var architectures []string
	var scanUrls []string
	if err == nil {
		indexManifest, err := index.IndexManifest()
		if err == nil {
			manifests := indexManifest.Manifests
			if manifests != nil {
				for _, manifest := range manifests {
					arch := manifest.Platform.Architecture
					if arch != "unknown" {
						architectures = append(architectures, arch)
						scanUrls = append(scanUrls, c.getScanUrl(ref.Context().Name(), manifest.Digest.String(), arch))
					}
				}
			}
		}
	}

	if architectures == nil {
		cf, err := i.ConfigFile()
		if err == nil {
			scanUrls = append(scanUrls, c.getScanUrl(ref.String(), "", cf.Architecture))
		}
	}
	inspectIcon, _ := icons.ReadFile("img/inspect-icon.svg")
	inspectUrl := template.HTML(fmt.Sprintf("<a href=%s/view/inspect?image=%s>%s</a>", waveServerUrl, ref, inspectIcon))

	scanUrl := template.HTML(strings.Join(scanUrls, ""))

	return registry.ImageInfo{Image: i, Reference: ref.String(), Architectures: architectures, ScanUrl: scanUrl, InspectUrl: inspectUrl}, nil
}

func (c *Registry) getScanUrl(ref string, digest string, platform string) string {
	waveServerUrl := c.cfg.WaveServerUrl
	scanIcon, _ := icons.ReadFile("img/scan-icon.svg")

	if !strings.HasPrefix(waveServerUrl, "https://") && !strings.HasPrefix(waveServerUrl, "http://") {
		waveServerUrl = "https://" + waveServerUrl
	}

	apiURL := fmt.Sprintf("%s/view/scans", waveServerUrl)

	imageRef := ref
	if digest != "" {
		imageRef = fmt.Sprintf("%s@%s", ref, digest)
	}

	script := fmt.Sprintf(
		`<svg onclick="fetchScan('%s', '%s')" xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-search" viewBox="0 0 16 16" style="cursor: pointer;">
				<title>%s</title> 
				%s</svg>`,
		apiURL, imageRef, platform, scanIcon,
	)

	return script
}

func New(rootCfg *cfg.Root) *Registry {
	cfg := config{
		Registry:      rootCfg.RegistryHostname,
		User:          rootCfg.RegistryUser,
		Password:      rootCfg.RegistryPassword,
		TLSEnabled:    rootCfg.TLSEnabled,
		SkipTLSVerify: rootCfg.SkipTLSVerify,
		WaveServerUrl: rootCfg.WaveServerUrl,
	}

	return &Registry{
		cfg: cfg,
	}
}
