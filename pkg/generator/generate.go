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
package generator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"

	"github.com/regclient/regclient"
	"github.com/seqeralabs/staticreg/pkg/filler"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/seqeralabs/staticreg/pkg/templates"
	"github.com/seqeralabs/staticreg/static"
)

type Generator struct {
	rc               *regclient.RegClient
	filler           *filler.Filler
	absoluteDir      string
	registryHostname string
	baseDir          string
}

func New(
	rc *regclient.RegClient,
	filler *filler.Filler,
	absoluteDir string,
	registryHostname string,
	baseDir string,
) *Generator {
	return &Generator{
		rc:               rc,
		absoluteDir:      absoluteDir,
		registryHostname: registryHostname,
		baseDir:          baseDir,
		filler:           filler,
	}
}

func (g *Generator) Generate(
	ctx context.Context) error {
	log := logger.FromContext(ctx)

	staticDir := path.Join(g.baseDir, "static")

	if err := os.MkdirAll(staticDir, 0755); err != nil {
		return err
	}

	styleCSSFile, err := os.Create(path.Join(staticDir, "style.css"))
	if err != nil {
		return err
	}
	err = static.RenderStyle(styleCSSFile)
	if err != nil {
		return err
	}
	defer styleCSSFile.Close()
	log.Info("generating repositories list page")
	indexFile, err := os.Create(path.Join(g.baseDir, "index.html"))
	if err != nil {
		return err
	}
	defer indexFile.Close()

	err = g.generateIndex(ctx, indexFile)
	if err != nil {
		return err
	}

	repos, err := g.rc.RepoList(ctx, g.registryHostname)
	if err != nil {
		return err
	}

	for _, repo := range repos.Repositories {
		repoLog := log.With(slog.String("repo", repo))
		repoLog.Info("generating repository page")
		repoDir := path.Join(g.baseDir, "repo", repo)
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			return err
		}
		f, err := os.Create(path.Join(repoDir, "index.html"))
		if err != nil {
			return err
		}
		defer f.Close()

		err = g.generateRepository(ctx, f, repo)
		if err != nil {
			repoLog.Error("error generating repository page, skipping...")
		}
	}

	return nil

}

func (g *Generator) generateRepository(
	ctx context.Context,
	w io.Writer,
	repo string,
) error {
	repoData, err := g.filler.RepoData(ctx, repo)
	if err != nil {
		return err
	}
	if repoData == nil {
		return fmt.Errorf("empty repo data")
	}
	err = templates.RenderRepository(w, *repoData)
	if err != nil {
		return err
	}
	return nil
}

func (g *Generator) generateIndex(
	ctx context.Context,
	w io.Writer,
) error {
	log := logger.FromContext(ctx)

	repositoriesData := []templates.RepositoryData{}

	baseData := g.filler.BaseData()
	repos, err := g.rc.RepoList(ctx, g.registryHostname)
	if err != nil {
		return err
	}

	for _, repo := range repos.Repositories {
		repoData, err := g.filler.RepoData(ctx, repo)
		if err != nil {
			log.Warn("could not retrieve repo data", slog.String("repo", repo), logger.ErrAttr(err))
		}
		if repoData == nil {
			continue
		}
		repositoriesData = append(repositoriesData, *repoData)
	}

	return templates.RenderIndex(w, templates.IndexData{
		BaseData:     baseData,
		Repositories: repositoriesData,
	})

}
