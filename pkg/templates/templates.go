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
package templates

import (
	"embed"
	_ "embed"
	"html/template"
	"io"
	"path"
)

//go:embed tmpl/*
var templates embed.FS

var htmlTemplates map[string]*template.Template

func init() {
	templateDefs := map[string]string{
		"index":      "index.html",
		"repository": "repository.html",
		"404":        "404.html",
		"500":        "500.html",
	}
	htmlTemplates = make(map[string]*template.Template, len(templateDefs))
	for tplName, templateDef := range templateDefs {
		tpl, err := template.New(templateDef).ParseFS(templates, path.Join("tmpl", templateDef))
		if err != nil {
			panic(err)
		}
		htmlTemplates[tplName] = tpl
	}
}

type BaseData struct {
	AbsoluteDir  string
	RegistryName string
	LastUpdated  string
}

type IndexData struct {
	BaseData
	Repositories []RepositoryData
}

func RenderIndex(w io.Writer, data IndexData) error {
	tpl := htmlTemplates["index"]
	return tpl.Execute(w, data)
}

type TagData struct {
	Name          string
	Tag           string
	PullReference string
	CreatedAt     string
}

type RepositoryData struct {
	BaseData
	RepositoryName string
	PullReference  string
	Tags           []TagData
	LastUpdatedAt  string
}

func RenderRepository(w io.Writer, data RepositoryData) error {
	tpl := htmlTemplates["repository"]
	return tpl.Execute(w, data)
}

func Render404(w io.Writer, data BaseData) error {
	tpl := htmlTemplates["404"]
	return tpl.Execute(w, data)
}

func Render500(w io.Writer, data BaseData) error {
	tpl := htmlTemplates["500"]
	return tpl.Execute(w, data)
}
