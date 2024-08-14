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
package async

import (
	"context"
	"fmt"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
	v1 "github.com/regclient/regclient/types/oci/v1"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/seqeralabs/staticreg/pkg/registry"
)

const imageInfoRequestsBufSize = 10
const tagRequestBufferSize = 10

// Async is a struct that wraps an underlying registry.Client
// to provide asynchronous methods for interacting with a container registry.
// It continuously syncs data from the registry in a separate goroutine.
type Async struct {
	// underlying is the actual registry client that does the registry operations, remember this is just a wrapper!
	underlying registry.Client

	repos []string

	repositoryTags *xsync.MapOf[string, []string]

	imageInfo *xsync.MapOf[imageInfoKey, imageInfo]

	repositoryRequestBuffer chan repositoryRequest
	imageInfoRequestsBuffer chan imageInfoRequest
}

type imageInfoKey struct {
	repo string
	tag  string
}
type repositoryRequest struct {
	repo string
}

type imageInfoRequest struct {
	repo string
	tag  string
}

type imageInfo struct {
	image     *v1.Image
	reference string
}

func (c *Async) Start(ctx context.Context) error {
	// TODO(fntlnz): maybe instead of errCh use a backoff and retry ops
	errCh := make(chan error, 1)
	go func() {
		// TODO(fntlnz):find a better strategy than waiting

		for {
			err := c.synchronizeRepositories(ctx)
			if err != nil {
				errCh <- err
			}
			time.Sleep(time.Minute)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-c.repositoryRequestBuffer:
				err := c.handleRepositoryRequest(ctx, req)
				if err != nil {
					errCh <- err
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-c.imageInfoRequestsBuffer:
				err := c.handleImageInfoRequest(ctx, req)
				if err != nil {
					errCh <- err
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

func (c *Async) synchronizeRepositories(ctx context.Context) error {
	repos, err := c.underlying.RepoList(ctx)
	if err != nil {
		return err
	}

	for _, r := range repos {
		c.repositoryRequestBuffer <- repositoryRequest{repo: r}
	}

	c.repos = repos
	return nil
}

func (c *Async) handleRepositoryRequest(ctx context.Context, req repositoryRequest) error {
	log := logger.FromContext(ctx)
	tags, err := c.underlying.TagList(ctx, req.repo)

	if err != nil {
		log.Warn("could not list tags for image", logger.ErrAttr(err))
		return nil

	}
	c.repositoryTags.Store(req.repo, tags)

	for _, t := range tags {
		c.imageInfoRequestsBuffer <- imageInfoRequest{
			repo: req.repo,
			tag:  t,
		}
	}

	return nil
}

func (c *Async) handleImageInfoRequest(ctx context.Context, req imageInfoRequest) error {
	log := logger.FromContext(ctx)
	key := imageInfoKey(req)
	i, r, err := c.underlying.ImageInfo(ctx, req.repo, req.tag)
	if err != nil {
		log.Warn("could not get image info for tag", logger.ErrAttr(err))
		return nil
	}
	imageInfo := imageInfo{
		image:     i,
		reference: r,
	}
	c.imageInfo.Store(key, imageInfo)
	return nil
}

func (c *Async) RepoList(ctx context.Context) ([]string, error) {
	return c.repos, nil
}

func (c *Async) TagList(ctx context.Context, repo string) ([]string, error) {
	tags, ok := c.repositoryTags.Load(repo)
	if !ok {
		return nil, fmt.Errorf("no tags found") // TODO(fntlnz): make an error var
	}
	return tags, nil
}

func (c *Async) ImageInfo(ctx context.Context, repo string, tag string) (image *v1.Image, reference string, err error) {
	key := imageInfoKey{
		repo: repo,
		tag:  tag,
	}
	info, ok := c.imageInfo.Load(key)
	if !ok {
		return nil, "", fmt.Errorf("image info not found") // TODO(fntlnz): make an error var
	}
	return info.image, info.reference, nil
}

func New(client registry.Client) *Async {
	return &Async{
		underlying:              client,
		repositoryTags:          xsync.NewMapOf[string, []string](),
		imageInfo:               xsync.NewMapOf[imageInfoKey, imageInfo](),
		repositoryRequestBuffer: make(chan repositoryRequest, tagRequestBufferSize),
		imageInfoRequestsBuffer: make(chan imageInfoRequest, imageInfoRequestsBufSize),
	}
}
