package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
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
	store := persistence.NewInMemoryStore(cacheDuration)
	r.Use(injectLoggerMiddleware(log))

	r.NoRoute(serverImpl.NoRouteHandler)
	r.Use(serverImpl.NotFoundHandler)

	r.GET("/static/style.css", serverImpl.CSSHandler)

	r.GET("/", cache.CachePage(store, cacheDuration, serverImpl.RepositoriesListHandler))
	r.GET("/repo/*slug", cache.CachePage(store, cacheDuration, serverImpl.RepositoryHandler))

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
