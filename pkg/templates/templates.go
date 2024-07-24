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
