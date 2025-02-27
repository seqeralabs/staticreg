package server

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	cache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	sloggin "github.com/samber/slog-gin"
	"github.com/seqeralabs/staticreg/pkg/static"
	"golang.org/x/sync/errgroup"

	"github.com/gin-gonic/gin"
)

const (
	robotsTxt = "User-agent: *\nDisallow: /\n"
)

var (
	robotsTxtETag = fmt.Sprintf("\"%x\"", sha256.Sum256([]byte(robotsTxt)))
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
		DefaultLevel:       slog.LevelDebug,
		WithUserAgent:      true,
		WithRequestID:      true,
		WithRequestBody:    false,
		WithResponseHeader: true,
		WithRequestHeader:  true,
	}

	r.Use(sloggin.NewWithConfig(log, lmConfig))
	r.Use(gin.Recovery())
	store := persist.NewMemoryStore(cacheDuration)
	r.Use(injectLoggerMiddleware(log))
	r.NoRoute(serverImpl.NoRouteHandler)
	r.Use(serverImpl.NotFoundHandler)
	r.Use(serverImpl.InternalServerErrorHandler)

	staticRouter := r.Group("/static")
	{
		staticRouter.Use(cacheControlMiddleware())
		staticRouter.StaticFS("/", http.FS(static.Assets))
	}

	ignoredUAMiddleware := ignoreUserAgentMiddleware(ignoredUserAgents)

	r.Use(ignoredUAMiddleware)
	r.GET("/robots.txt", robotsTxtHandler)
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

func (s *Server) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(s.server.ListenAndServe)
	g.Go(func() error {
		<-ctx.Done()
		return s.server.Shutdown(context.Background())
	})
	return g.Wait()
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

func cacheControlMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "public, max-age=604800, immutable")
	}
}

func robotsTxtHandler(ctx *gin.Context) {
	ctx.Header("ETag", robotsTxtETag)
	ctx.Header("Content-Type", "text/plain")
	match := ctx.GetHeader("If-None-Match")
	if len(match) > 0 && match == robotsTxtETag {
		ctx.Status(http.StatusNotModified)
		return
	}
	ctx.String(http.StatusOK, robotsTxt)
}
