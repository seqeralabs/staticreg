package server

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	sloggin "github.com/samber/slog-gin"

	// "github.com/chenyahui/gin-cache/persist"
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
	ignoredUserAgents []string,
) (*Server, error) {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	lmConfig := sloggin.Config{
		WithUserAgent:      true,
		WithRequestID:      true,
		WithRequestBody:    false,
		WithResponseHeader: true,
		WithRequestHeader:  true,
	}

	r.Use(sloggin.NewWithConfig(log, lmConfig))
	r.Use(gin.Recovery())
	// store := persist.NewMemoryStore(cacheDuration)
	r.Use(injectLoggerMiddleware(log))
	r.NoRoute(serverImpl.NoRouteHandler)
	r.Use(serverImpl.NotFoundHandler)
	r.Use(serverImpl.InternalServerErrorHandler)

	r.GET("/static/style.css", serverImpl.CSSHandler)

	ignoredUAMiddleware := ignoreUserAgentMiddleware(ignoredUserAgents)

	r.Use(ignoredUAMiddleware)
	htmlRoutes := r.Group("/")
	{
		// r.GET("/", cache.CacheByRequestURI(store, cacheDuration), serverImpl.RepositoriesListHandler)
		r.GET("/", serverImpl.RepositoriesListHandler)
		r.GET("/repo/*slug", serverImpl.RepositoryHandler)
		// r.GET("/repo/*slug", cache.CacheByRequestURI(store, cacheDuration), serverImpl.RepositoryHandler)
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

func ignoreUserAgentMiddleware(ignoredUserAgents []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userAgent := c.Request.UserAgent()
		for _, ignored := range ignoredUserAgents {
			if strings.Contains(userAgent, ignored) {
				c.Status(http.StatusOK)
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
