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
	"path"
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
	s.HierarchicalBrowseHandler(c)
}

func (s *StaticregServer) HierarchicalBrowseHandler(c *gin.Context) {
	baseData := s.dataFiller.BaseDataFromContext(c)
	currentPath := strings.TrimPrefix(c.Param("path"), "/")
	currentPath = strings.TrimSuffix(currentPath, "/")

	repos, err := s.regClient.RepoList(c)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	hierarchy, repositories := s.buildHierarchy(repos, currentPath, baseData)
	breadcrumbs := s.buildBreadcrumbs(currentPath)

	var buf bytes.Buffer
	err = templates.RenderIndex(&buf, templates.IndexData{
		BaseData:     baseData,
		CurrentPath:  currentPath,
		Breadcrumbs:  breadcrumbs,
		Nodes:        hierarchy,
		Repositories: repositories,
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

func (s *StaticregServer) buildHierarchy(repos map[string]registry.RepoData, currentPath string, baseData templates.BaseData) ([]*templates.HierarchicalNode, []templates.IndexRepositoryData) {
	folders := make(map[string]*templates.HierarchicalNode)
	var repositories []templates.IndexRepositoryData

	sortedRepos := make([]string, 0, len(repos))
	for k := range repos {
		sortedRepos = append(sortedRepos, k)
	}
	sort.Strings(sortedRepos)

	for _, repoName := range sortedRepos {
		repo := repos[repoName]
		parts := strings.Split(repoName, "/")

		if currentPath == "" {
			if len(parts) == 1 {
				repositories = append(repositories, templates.IndexRepositoryData{
					BaseData:       baseData,
					RepositoryName: repo.Name,
					PullReference:  repo.PullReference,
					LastUpdatedAt:  repo.LastUpdatedAt.Format(time.RFC3339),
				})
			} else {
				firstPart := parts[0]
				if folders[firstPart] == nil {
					folders[firstPart] = &templates.HierarchicalNode{
						Name:     firstPart,
						IsFolder: true,
					}
				}
			}
		} else {
			pathParts := strings.Split(currentPath, "/")
			if len(parts) > len(pathParts) && strings.HasPrefix(repoName, currentPath+"/") {
				remainingPath := strings.TrimPrefix(repoName, currentPath+"/")
				remainingParts := strings.Split(remainingPath, "/")

				if len(remainingParts) == 1 {
					repositories = append(repositories, templates.IndexRepositoryData{
						BaseData:       baseData,
						RepositoryName: repo.Name,
						PullReference:  repo.PullReference,
						LastUpdatedAt:  repo.LastUpdatedAt.Format(time.RFC3339),
					})
				} else {
					firstPart := remainingParts[0]
					if folders[firstPart] == nil {
						folders[firstPart] = &templates.HierarchicalNode{
							Name:     firstPart,
							IsFolder: true,
						}
					}
				}
			}
		}
	}

	var nodes []*templates.HierarchicalNode
	for _, folder := range folders {
		nodes = append(nodes, folder)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	return nodes, repositories
}

func (s *StaticregServer) buildBreadcrumbs(currentPath string) []string {
	if currentPath == "" {
		return []string{}
	}
	parts := strings.Split(currentPath, "/")
	breadcrumbs := make([]string, len(parts))
	copy(breadcrumbs, parts)
	return breadcrumbs
}

func (s *StaticregServer) RepositoryHandler(c *gin.Context) {

	slug := c.Param("slug")

	if len(slug) == 1 {
		_ = c.AbortWithError(http.StatusNotFound, servererrors.ErrSlugTooShort)
		return
	}

	slug = strings.TrimLeft(slug, "/")
	query := c.Query("q")

	repoData, err := s.dataFiller.RepoDataWithQuery(c, slug, query)
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

	// Add breadcrumbs to repository data
	repoData.Breadcrumbs = s.buildBreadcrumbs(slug)

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
	baseData := s.dataFiller.BaseDataFromContext(c)

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

	baseData := s.dataFiller.BaseDataFromContext(c)

	err := templates.Render500(c.Writer, baseData)
	if err != nil {
		c.Error(err)
	}

	log.Error("internal server error", slog.Any("errors", c.Errors))
}

type SearchResult struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	PullReference string `json:"pullReference"`
	LastUpdatedAt string `json:"lastUpdatedAt"`
	Type          string `json:"type"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}

func (s *StaticregServer) SearchHandler(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	repos, err := s.regClient.RepoList(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var results []SearchResult
	query = strings.ToLower(query)

	for repoName, repo := range repos {
		if strings.Contains(strings.ToLower(repoName), query) {
			results = append(results, SearchResult{
				Name:          repo.Name,
				Path:          path.Join("/repo/", repo.Name),
				PullReference: repo.PullReference,
				LastUpdatedAt: repo.LastUpdatedAt.Format(time.RFC3339),
				Type:          "repository",
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	c.JSON(http.StatusOK, SearchResponse{
		Results: results,
		Total:   len(results),
	})
}

func (s *StaticregServer) SearchResultsHandler(c *gin.Context) {
	query := c.Query("q")
	baseData := s.dataFiller.BaseDataFromContext(c)

	searchData := struct {
		templates.BaseData
		Query   string
		Results []SearchResult
		Total   int
	}{
		BaseData: baseData,
		Query:    query,
		Results:  []SearchResult{},
		Total:    0,
	}

	if query != "" {
		repos, err := s.regClient.RepoList(c)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		queryLower := strings.ToLower(query)
		for repoName, repo := range repos {
			if strings.Contains(strings.ToLower(repoName), queryLower) {
				searchData.Results = append(searchData.Results, SearchResult{
					Name:          repo.Name,
					Path:          "/repo/" + repo.Name,
					PullReference: repo.PullReference,
					LastUpdatedAt: repo.LastUpdatedAt.Format(time.RFC3339),
					Type:          "repository",
				})
			}
		}

		sort.Slice(searchData.Results, func(i, j int) bool {
			return searchData.Results[i].Name < searchData.Results[j].Name
		})
		searchData.Total = len(searchData.Results)
	}

	var buf bytes.Buffer
	err := templates.RenderSearch(&buf, searchData)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
	_, err = buf.WriteTo(c.Writer)
	if err != nil {
		_ = c.Error(err)
		return
	}
}

func (s *StaticregServer) NoRouteHandler(c *gin.Context) {
	originalPath := c.Request.URL.Path
	if strings.HasPrefix(originalPath, "/repo/") {
		baseData := s.dataFiller.BaseDataFromContext(c)
		err := templates.Render404(c.Writer, baseData)
		if err != nil {
			_ = c.Error(err)
			return
		}
	}

	repoPath := path.Join("/repo", originalPath)
	if c.Request.URL.RawQuery != "" {
		repoPath += "?" + c.Request.URL.RawQuery
	}
	c.Redirect(http.StatusMovedPermanently, repoPath)
}
