package templates

import (
	"embed"
	_ "embed"
	"io"
	"text/template"
)

//go:embed tmpl/*
var templates embed.FS

type IndexData struct {
	RegistryName string
	Repositories []string
}

func RenderIndex(w io.Writer, data IndexData) error {
	tpl, err := template.New("index.html").ParseFS(templates, "tmpl/index.html")
	if err != nil {
		return err
	}
	return tpl.Execute(w, data)
}

type TagData struct {
	Name      string
	Tag       string
	CreatedAt string
}

type RepositoryData struct {
	RegistryName   string
	RepositoryName string
	Tags           []TagData
}

func RenderRepository(w io.Writer, data RepositoryData) error {
	tpl, err := template.New("repository.html").ParseFS(templates, "tmpl/repository.html")
	if err != nil {
		return err
	}
	return tpl.Execute(w, data)
}
