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
	"errors"
	"log/slog"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/puzpuzpuz/xsync/v3"
	"golang.org/x/sync/errgroup"

	"github.com/cenkalti/backoff/v4"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/seqeralabs/staticreg/pkg/registry"
)

const imageInfoRequestsBufSize = 10
const tagRequestBufferSize = 10

var (
	ErrNoTagsFound       = errors.New("no tags found")
	ErrImageInfoNotFound = errors.New("image info not found")
)

// Async is a struct that wraps an underlying registry.Client
// to provide asynchronous methods for interacting with a container registry.
// It continuously syncs data from the registry in a separate goroutine.
type Async struct {
	// underlying is the actual registry client that does the registry operations, remember this is just a wrapper!
	underlying registry.Client
	// refreshInterval represents the time to wait to synchronize repositories again after a successful synchronization
	refreshInterval time.Duration

	// repos is an in memory list of all the repository names in the registry
	repos []string

	// repositoryTags represents the list of tags for each repository
	repositoryTags *xsync.MapOf[string, []string]

	// imageInfo contains the image information indexed by repo name and tag
	imageInfo *xsync.MapOf[imageInfoKey, imageInfo]
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
	image     v1.Image
	reference string
}

func (c *Async) Start(ctx context.Context) error {
	log := logger.FromContext(ctx)
	g, ctx := errgroup.WithContext(ctx)

	// repositoryRequestBuffer generates requests for the `handleRepositoryRequest`
	// handler that is responsible for retrieving the tags for a given image and
	// scheduling new jobs on `imageInfoRequestsBuffer`
	repositoryRequestBuffer := make(chan repositoryRequest, tagRequestBufferSize)
	// imageInfoRequestsBuffer is responsible for feeding `handleImageInfoRequest`
	// so that image info is retrieved for each <repo,tag> combination
	imageInfoRequestsBuffer := make(chan imageInfoRequest, imageInfoRequestsBufSize)

	defer func() {
		close(repositoryRequestBuffer)
		close(imageInfoRequestsBuffer)
	}()

	g.Go(func() error {
		for {
			err := backoff.Retry(func() error {
				err := c.synchronizeRepositories(ctx, repositoryRequestBuffer)
				if err != nil {
					log.Error("err", logger.ErrAttr(err))
				}
				return err
			}, backoff.WithContext(newExponentialBackoff(), ctx))

			if err != nil {
				return err
			}

			wait := time.After(c.refreshInterval)

			select {
			case <-wait:
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req := <-repositoryRequestBuffer:
				c.handleRepositoryRequest(ctx, imageInfoRequestsBuffer, req)
			}
		}
	})

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req := <-imageInfoRequestsBuffer:
				c.handleImageInfoRequest(ctx, req)
			}
		}
	})

	return g.Wait()
}

func (c *Async) synchronizeRepositories(ctx context.Context, reqChan chan<- repositoryRequest) error {
	log := logger.FromContext(ctx)
	log.Info("starting process to synchronize repositories")
	repos, err := c.underlying.RepoList(ctx)
	if err != nil {
		return err
	}
	c.repos = repos

	for _, r := range repos {
		select {
		case reqChan <- repositoryRequest{repo: r}:
		case <-ctx.Done():
			return nil
		}
	}

	return nil
}

func (c *Async) handleRepositoryRequest(ctx context.Context, reqChan chan<- imageInfoRequest, req repositoryRequest) {
	log := logger.FromContext(ctx)
	reqLog := log.With(slog.Any("req", req))
	reqLog.Debug("handleRepositoryRequest")
	tags, err := c.underlying.TagList(ctx, req.repo)

	if err != nil {
		reqLog.Warn("could not list tags for image", logger.ErrAttr(err))
		return

	}

	c.repositoryTags.Store(req.repo, tags)

	for _, t := range tags {
		select {
		case reqChan <- imageInfoRequest{
			repo: req.repo,
			tag:  t,
		}:
		case <-ctx.Done():
			return
		}
	}
}

func (c *Async) handleImageInfoRequest(ctx context.Context, req imageInfoRequest) {
	log := logger.FromContext(ctx)
	reqLog := log.With(slog.Any("req", req))
	reqLog.Debug("handleImageInfoRequest")
	key := imageInfoKey(req)
	i, r, err := c.underlying.ImageInfo(ctx, req.repo, req.tag)
	if err != nil {
		reqLog.Warn("could not get image info for tag", logger.ErrAttr(err))
		return
	}
	imageInfo := imageInfo{
		image:     i,
		reference: r,
	}
	c.imageInfo.Store(key, imageInfo)
}

func (c *Async) RepoList(ctx context.Context) ([]string, error) {
	return c.repos, nil
}

// TagList contains
func (c *Async) TagList(ctx context.Context, repo string) ([]string, error) {
	tags, ok := c.repositoryTags.Load(repo)
	if !ok {
		return nil, ErrNoTagsFound
	}
	return tags, nil
}

func (c *Async) ImageInfo(ctx context.Context, repo string, tag string) (image v1.Image, reference string, err error) {
	key := imageInfoKey{
		repo: repo,
		tag:  tag,
	}
	info, ok := c.imageInfo.Load(key)
	if !ok {
		return nil, "", ErrImageInfoNotFound
	}
	return info.image, info.reference, nil
}

func New(client registry.Client, refreshInterval time.Duration) *Async {
	return &Async{
		underlying:      client,
		refreshInterval: refreshInterval,
		repositoryTags:  xsync.NewMapOf[string, []string](),
		imageInfo:       xsync.NewMapOf[imageInfoKey, imageInfo](),
	}
}

func newExponentialBackoff() *backoff.ExponentialBackOff {
	bo := backoff.NewExponentialBackOff()
	bo.Multiplier = 1.1
	bo.MaxInterval = 10 * time.Second
	bo.MaxElapsedTime = 5 * time.Minute
	return bo
}
