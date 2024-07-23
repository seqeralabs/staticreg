package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Server struct {
	server *http.Server
	gin    *gin.Engine
}

type ServerImpl interface {
	RepositoriesListHandler(ctx *gin.Context)
	RepositoryHandler(ctx *gin.Context)
}

func New(bindAddr string, serverImpl ServerImpl, log *slog.Logger) (*Server, error) {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	r.Use(injectLoggerMiddleware(log))

	r.GET("/", serverImpl.RepositoriesListHandler)
	r.GET("/repo/*slug", serverImpl.RepositoryHandler)

	srv := &http.Server{
		Handler: r,
		Addr:    bindAddr,
	}

	return &Server{
		gin:    r,
		server: srv,
	}, nil
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func injectLoggerMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("logger", log)
		c.Next()
	}
}
