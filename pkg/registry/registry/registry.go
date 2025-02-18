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
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/seqeralabs/staticreg/pkg/cfg"
	"github.com/seqeralabs/staticreg/pkg/registry"
	"strings"
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
	inspectUrl := fmt.Sprintf("%s/view/inspect?image=%s", waveServerUrl, ref)

	return registry.ImageInfo{Image: i, Reference: ref.String(), Architectures: architectures, ScanUrls: scanUrls, InspectUrl: inspectUrl}, nil
}

func (c *Registry) getScanUrl(ref string, digest string, platform string) string {

	waveServerUrl := c.cfg.WaveServerUrl

	if !strings.Contains(waveServerUrl, "https://") && !strings.Contains(waveServerUrl, "http://") {
		waveServerUrl = "https://" + waveServerUrl
	}

	const scanSvg = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-shield-check" viewBox="0 0 16 16">
						<path d="M5.338 1.59a61 61 0 0 0-2.837.856.48.48 0 0 0-.328.39c-.554 4.157.726 7.19 2.253 9.188a10.7 10.7 0 0 0 2.287 2.233c.346.244.652.42.893.533q.18.085.293.118a1 1 0 0 0 .101.025 1 1 0 0 0 .1-.025q.114-.034.294-.118c.24-.113.547-.29.893-.533a10.7 10.7 0 0 0 2.287-2.233c1.527-1.997 2.807-5.031 2.253-9.188a.48.48 0 0 0-.328-.39c-.651-.213-1.75-.56-2.837-.855C9.552 1.29 8.531 1.067 8 1.067c-.53 0-1.552.223-2.662.524zM5.072.56C6.157.265 7.31 0 8 0s1.843.265 2.928.56c1.11.3 2.229.655 2.887.87a1.54 1.54 0 0 1 1.044 1.262c.596 4.477-.787 7.795-2.465 9.99a11.8 11.8 0 0 1-2.517 2.453 7 7 0 0 1-1.048.625c-.28.132-.581.24-.829.24s-.548-.108-.829-.24a7 7 0 0 1-1.048-.625 11.8 11.8 0 0 1-2.517-2.453C1.928 10.487.545 7.169 1.141 2.692A1.54 1.54 0 0 1 2.185 1.43 63 63 0 0 1 5.072.56"/>
						<path d="M10.854 5.146a.5.5 0 0 1 0 .708l-3 3a.5.5 0 0 1-.708 0l-1.5-1.5a.5.5 0 1 1 .708-.708L7.5 7.793l2.646-2.647a.5.5 0 0 1 .708 0"/>
					</svg>`

	url := fmt.Sprintf("<a href=%s/view/scans?image=%s title=%s>%s</a>", waveServerUrl, ref, platform, scanSvg)
	if digest != "" {
		url = fmt.Sprintf("<a href=%s/view/scans?image=%s@%s title=%s>%s</a>", waveServerUrl, ref, digest, platform, scanSvg)
	}
	return url
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
