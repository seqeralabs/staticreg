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
package staticreg

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/seqeralabs/staticreg/pkg/filler"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/seqeralabs/staticreg/pkg/registry"
	"github.com/seqeralabs/staticreg/pkg/registry/errs"
	"github.com/seqeralabs/staticreg/pkg/templates"

	servererrors "github.com/seqeralabs/staticreg/pkg/server/errors"
)

type StaticregServer struct {
	regClient        registry.Client
	dataFiller       *filler.Filler
	registryHostname string
}

func New(
	regClient registry.Client,
	dataFiller *filler.Filler,
	registryHostname string,
) *StaticregServer {
	return &StaticregServer{
		regClient:        regClient,
		dataFiller:       dataFiller,
		registryHostname: registryHostname,
	}
}

func (s *StaticregServer) RepositoriesListHandler(c *gin.Context) {
	repositoriesData := []templates.IndexRepositoryData{}
	baseData := s.dataFiller.BaseData()

	repos, err := s.regClient.RepoList(c)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	sortedRepos := make([]string, len(repos))
	i := 0
	for k := range repos {
		sortedRepos[i] = k
		i++
	}
	sort.Strings(sortedRepos)

	for _, rk := range sortedRepos {
		repo, ok := repos[rk]
		if !ok {
			continue
		}
		idata := templates.IndexRepositoryData{
			BaseData:       baseData,
			RepositoryName: repo.Name,
			PullReference:  repo.PullReference,
			LastUpdatedAt:  repo.LastUpdatedAt.Format(time.RFC3339),
		}
		repositoriesData = append(repositoriesData, idata)
	}

	var buf bytes.Buffer
	err = templates.RenderIndex(&buf, templates.IndexData{
		BaseData:     baseData,
		Repositories: repositoriesData,
	})

	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
	_, err = buf.WriteTo(c.Writer)
	if err != nil {
		c.Error(err)
		return
	}
}

func (s *StaticregServer) RepositoryHandler(c *gin.Context) {

	slug := c.Param("slug")

	if len(slug) == 1 {
		_ = c.AbortWithError(http.StatusNotFound, servererrors.ErrSlugTooShort)
		return
	}

	slug = strings.TrimLeft(slug, "/")

	repoData, err := s.dataFiller.RepoData(c, slug)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidReference) {
			_ = c.AbortWithError(http.StatusNotFound, err)
			return
		}
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if repoData == nil {
		_ = c.AbortWithError(http.StatusNotFound, servererrors.ErrRepositoryNotFound)
		return
	}

	var buf bytes.Buffer
	err = templates.RenderRepository(&buf, *repoData)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
	_, err = buf.WriteTo(c.Writer)
	if err != nil {
		c.Error(err)
		return
	}
}

func (s *StaticregServer) NotFoundHandler(c *gin.Context) {
	c.Next()
	if len(c.Errors) == 0 {
		return
	}

	if c.Writer.Status() != http.StatusNotFound {
		return
	}
	baseData := s.dataFiller.BaseData()

	var buf bytes.Buffer
	err := templates.Render404(&buf, baseData)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	_, err = buf.WriteTo(c.Writer)
	if err != nil {
		c.Error(err)
		return
	}
}

func (s *StaticregServer) InternalServerErrorHandler(c *gin.Context) {
	c.Next()
	if len(c.Errors) == 0 {
		return
	}
	log := logger.FromContext(c)

	if len(c.Errors) > 0 &&
		c.Writer.Status() != http.StatusInternalServerError &&
		c.Writer.Status() != http.StatusNotFound &&
		c.Writer.Status() != http.StatusBadRequest {
		log.Error("handler error without error status code", slog.Any("errors", c.Errors))
		return
	}

	if c.Writer.Status() != http.StatusInternalServerError {
		return
	}

	baseData := s.dataFiller.BaseData()

	err := templates.Render500(c.Writer, baseData)
	if err != nil {
		c.Error(err)
	}

	log.Error("internal server error", slog.Any("errors", c.Errors))
}

func (s *StaticregServer) NoRouteHandler(c *gin.Context) {
	baseData := s.dataFiller.BaseData()

	err := templates.Render404(c.Writer, baseData)
	if err != nil {
		c.Error(err)
		return
	}
}
