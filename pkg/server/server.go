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
	"github.com/seqeralabs/staticreg/pkg/registry/async"
	"github.com/seqeralabs/staticreg/pkg/serviceinfo"
	"github.com/seqeralabs/staticreg/pkg/static"
	"github.com/seqeralabs/staticreg/pkg/webhook"
	"golang.org/x/sync/errgroup"

	"github.com/gin-gonic/gin"
)

const (
	robotsTxt = "User-agent: *\nDisallow: /\n"
)

var (
	robotsTxtETag = fmt.Sprintf("\"%x\"", sha256.Sum256([]byte(robotsTxt)))
)

type WebhookService interface {
	InvalidateRepository(repository string) error
	SavePullEvent(ctx context.Context, event *webhook.DistributionEvent) error
}

type Server struct {
	server         *http.Server
	gin            *gin.Engine
	cacheManager   *CacheManager
	webhookService WebhookService
}

type ServerImpl interface {
	RepositoriesListHandler(ctx *gin.Context)
	HierarchicalBrowseHandler(ctx *gin.Context)
	RepositoryHandler(ctx *gin.Context)
	SearchHandler(ctx *gin.Context)
	SearchResultsHandler(ctx *gin.Context)
	NotFoundHandler(ctx *gin.Context)
	NoRouteHandler(ctx *gin.Context)
	InternalServerErrorHandler(ctx *gin.Context)
}

func New(
	bindAddr string,
	serverImpl ServerImpl,
	asyncRegistry *async.Async,
	log *slog.Logger,
	cacheDuration time.Duration,
	ignoredUserAgents []string,
) (*Server, error) {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	si := serviceinfo.New()

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
	cacheManager := NewCacheManager(store, asyncRegistry, log)
	whService := webhook.NewServiceAdapter(cacheManager, log)

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
	r.GET("/service-info", serviceInfoHandler(si))

	apiRoutes := r.Group("/api")
	{
		apiRoutes.GET("/search", serverImpl.SearchHandler)
		apiRoutes.POST("/webhook/registry", registryWebhookHandler(whService, log))
	}

	htmlRoutes := r.Group("/")
	{
		r.GET("/", cache.CacheByRequestURI(store, cacheDuration), serverImpl.RepositoriesListHandler)
		r.GET("/browse/*path", cache.CacheByRequestURI(store, cacheDuration), serverImpl.HierarchicalBrowseHandler)
		r.GET("/repo/*slug", cache.CacheByRequestURI(store, cacheDuration), serverImpl.RepositoryHandler)
		r.GET("/search", serverImpl.SearchResultsHandler)
	}
	htmlRoutes.Use(htmlContentTypeMiddleware)

	srv := &http.Server{
		Handler: r,
		Addr:    bindAddr,
	}

	return &Server{
		gin:            r,
		server:         srv,
		cacheManager:   cacheManager,
		webhookService: whService,
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

func serviceInfoHandler(si *serviceinfo.ServiceInfo) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, si)
	}
}

func registryWebhookHandler(whService WebhookService, log *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var envelope webhook.DistributionEventEnvelope
		if err := ctx.ShouldBindJSON(&envelope); err != nil {
			log.Warn("Failed to parse webhook payload", "error", err)
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
			return
		}

		processedRepos := make(map[string]bool)
		eventsProcessed := 0

		for _, event := range envelope.Events {

			// 1. Handle PUSH events
			if event.IsManifestPush() {
				repository := event.Target.Repository

				// Avoid duplicate processing for the same repository in this batch
				if processedRepos[repository] {
					log.Debug("Repository already processed in this batch", "repository", repository)
					eventsProcessed++
					continue
				}

				log.Info("Processing push event for cache invalidation",
					"repository", repository,
					"digest", event.Target.Digest,
					"tag", event.Target.Tag)

				err := whService.InvalidateRepository(repository)
				if err != nil {
					log.Error("Failed to invalidate repository cache",
						"repository", repository,
						"error", err)
				} else {
					processedRepos[repository] = true
				}
				eventsProcessed++
				continue
			}

			// 2. Handle PULL events
			if event.IsManifestPull() {
				if err := whService.SavePullEvent(ctx, &event); err != nil {
					log.Error("Failed to save pull event to database",
						"repository", event.Target.Repository,
						"error", err)
				}
				eventsProcessed++
				continue
			}

			log.Debug("Skipping unhandled event action",
				"action", event.Action,
				"mediaType", event.Target.MediaType,
				"repository", event.Target.Repository)
			eventsProcessed++
		}

		log.Info("Webhook processing completed",
			"totalEvents", len(envelope.Events),
			"eventsProcessed", eventsProcessed,
			"repositoriesInvalidated", len(processedRepos))

		ctx.JSON(http.StatusOK, gin.H{
			"message":                 "Webhook processed successfully",
			"eventsProcessed":         eventsProcessed,
			"repositoriesInvalidated": len(processedRepos),
		})
	}
}
