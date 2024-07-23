package staticreg

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/errs"
	"github.com/seqeralabs/staticreg/pkg/filler"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	"github.com/seqeralabs/staticreg/pkg/templates"
	"github.com/seqeralabs/staticreg/static"

	servererrors "github.com/seqeralabs/staticreg/pkg/server/errors"
)

type StaticregServer struct {
	rc               *regclient.RegClient
	dataFiller       *filler.Filler
	registryHostname string
}

func New(
	rc *regclient.RegClient,
	dataFiller *filler.Filler,
	registryHostname string,
) *StaticregServer {
	return &StaticregServer{
		rc:               rc,
		dataFiller:       dataFiller,
		registryHostname: registryHostname,
	}
}

func (s *StaticregServer) RepositoriesListHandler(c *gin.Context) {
	log := logger.FromContext(c)

	repositoriesData := []templates.RepositoryData{}
	baseData := templates.BaseData{
		AbsoluteDir:  "/",
		RegistryName: s.registryHostname,
	}

	repos, err := s.rc.RepoList(c, s.registryHostname)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	for _, repo := range repos.Repositories {
		repoData, err := s.dataFiller.RepoData(c, repo)
		if err != nil {
			log.Warn("could not retrieve repo data", slog.String("repo", repo), logger.ErrAttr(err))
		}
		repositoriesData = append(repositoriesData, *repoData)
	}

	err = templates.RenderIndex(c.Writer, templates.IndexData{
		BaseData:     baseData,
		Repositories: repositoriesData,
	})

	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
}

func (s *StaticregServer) RepositoryHandler(c *gin.Context) {

	slug := c.Param("slug")

	if len(slug) == 1 {
		_ = c.AbortWithError(http.StatusNotFound, servererrors.ErrRepositoryNotFound)
		return
	}

	slug = strings.TrimLeft(slug, "/")

	repoData, err := s.dataFiller.RepoData(c, slug)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidReference) {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if repoData == nil {
		_ = c.AbortWithError(http.StatusNotFound, servererrors.ErrRepositoryNotFound)
		return
	}

	err = templates.RenderRepository(c.Writer, *repoData)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)

}

func (s *StaticregServer) NotFoundHandler(c *gin.Context) {
	c.Next()
	if len(c.Errors) == 0 {
		return
	}

	baseData := s.dataFiller.BaseData()
	err := templates.Render404(c.Writer, baseData)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func (s *StaticregServer) NoRouteHandler(c *gin.Context) {
	baseData := s.dataFiller.BaseData()
	err := templates.Render404(c.Writer, baseData)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func (s *StaticregServer) CSSHandler(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/css")
	err := static.RenderStyle(c.Writer)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}
