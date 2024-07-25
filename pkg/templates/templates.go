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
	"io"
	"text/template"
)

//go:embed tmpl/*
var templates embed.FS

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
	tpl, err := template.New("index.html").ParseFS(templates, "tmpl/index.html")
	if err != nil {
		return err
	}
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
	tpl, err := template.New("repository.html").ParseFS(templates, "tmpl/repository.html")
	if err != nil {
		return err
	}
	return tpl.Execute(w, data)
}

func Render404(w io.Writer, data BaseData) error {
	tpl, err := template.New("404.html").ParseFS(templates, "tmpl/404.html")
	if err != nil {
		return err
	}
	return tpl.Execute(w, data)
}

func Render500(w io.Writer, data BaseData) error {
	tpl, err := template.New("500.html").ParseFS(templates, "tmpl/500.html")
	if err != nil {
		return err
	}
	return tpl.Execute(w, data)
}
