package server

import (
	cache "github.com/chenyahui/gin-cache"
	"log/slog"
	"net/http"
	"time"

	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-gonic/gin"
)

type Server struct {
	server *http.Server
	gin    *gin.Engine
}

type ServerImpl interface {
	RepositoriesListHandler(ctx *gin.Context)
	RepositoryHandler(ctx *gin.Context)
	NotFoundHandler(ctx *gin.Context)
	NoRouteHandler(ctx *gin.Context)
	InternalServerErrorHandler(ctx *gin.Context)
	CSSHandler(ctx *gin.Context)
}

func New(
	bindAddr string,
	serverImpl ServerImpl,
	log *slog.Logger,
	cacheDuration time.Duration,
) (*Server, error) {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	store := persist.NewMemoryStore(cacheDuration)
	r.Use(injectLoggerMiddleware(log))
	r.NoRoute(serverImpl.NoRouteHandler)
	r.Use(serverImpl.NotFoundHandler)
	r.Use(serverImpl.InternalServerErrorHandler)

	r.GET("/static/style.css", serverImpl.CSSHandler)

	htmlRoutes := r.Group("/")
	{
		r.GET("/", cache.CacheByRequestURI(store, cacheDuration), serverImpl.RepositoriesListHandler)
		r.GET("/repo/*slug", cache.CacheByRequestURI(store, cacheDuration), serverImpl.RepositoryHandler)
	}
	htmlRoutes.Use(htmlContentTypeMiddleware)

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

func htmlContentTypeMiddleware(ctx *gin.Context) {
	ctx.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
}
