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

	"errors"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	regclienterrs "github.com/regclient/regclient/types/errs"
	v1 "github.com/regclient/regclient/types/oci/v1"
	"github.com/regclient/regclient/types/ref"
	"github.com/seqeralabs/staticreg/pkg/cfg"
	"github.com/seqeralabs/staticreg/pkg/registry/errs"
)

func hostFromConfig(rootCfg *cfg.Root) config.Host {
	regHost := config.Host{
		Name:     rootCfg.RegistryHostname,
		Hostname: rootCfg.RegistryHostname,
		User:     rootCfg.RegistryUser,
		Pass:     rootCfg.RegistryPassword,
	}

	if !rootCfg.TLSEnabled {
		regHost.TLS = config.TLSDisabled
	}

	if rootCfg.SkipTLSVerify {
		regHost.TLS = config.TLSInsecure
	}

	return regHost
}

type Registry struct {
	regHost       config.Host
	catalogClient *regclient.RegClient
	pullClient    *regclient.RegClient
}

func (c *Registry) RepoList(ctx context.Context) ([]string, error) {
	ret, err := c.catalogClient.RepoList(ctx, c.regHost.Hostname)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return []string{}, nil
	}
	return ret.Repositories, nil
}

func (c *Registry) TagList(ctx context.Context, repo string) ([]string, error) {
	repoFullRef := fmt.Sprintf("%s/%s", c.regHost.Hostname, repo)
	repoRef, err := ref.New(repoFullRef)
	if err != nil {
		if errors.Is(regclienterrs.ErrInvalidReference, err) {
			return nil, errs.ErrInvalidReference
		}
		return nil, err
	}
	ret, err := c.pullClient.TagList(ctx, repoRef)
	if err != nil {
		if errors.Is(regclienterrs.ErrNotFound, err) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	if ret == nil {
		return []string{}, nil
	}
	return ret.Tags, nil
}

func (c *Registry) ImageInfo(ctx context.Context, repo string, tag string) (image *v1.Image, reference string, err error) {

	tagFullRef := fmt.Sprintf("%s/%s:%s", c.regHost.Hostname, repo, tag)
	tagRef, err := ref.New(tagFullRef)
	if err != nil {
		if errors.Is(regclienterrs.ErrInvalidReference, err) {
			return nil, "", errs.ErrInvalidReference
		}
		return nil, "", err
	}

	i, err := c.pullClient.ImageConfig(ctx, tagRef)
	if err != nil {
		if errors.Is(regclienterrs.ErrNotFound, err) {
			return nil, "", errs.ErrNotFound
		}
		return nil, "", err
	}
	img := i.GetConfig()
	return &img, tagRef.CommonName(), err
}

func New(rootCfg *cfg.Root) *Registry {
	regHost := hostFromConfig(rootCfg)
	catalogClient := regclient.New(
		regclient.WithConfigHost(regHost),
		regclient.WithUserAgent("seqera/staticreg"),
	)
	pullClient := regclient.New(
		regclient.WithConfigHost(regHost),
		regclient.WithUserAgent("seqera/staticreg"),
	)
	return &Registry{
		catalogClient: catalogClient,
		pullClient:    pullClient,
		regHost:       regHost,
	}
}
