package main

import (
	"context"
	"net/http"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/RodolfoBonis/go-otel-agent/fxmodule"
	"github.com/RodolfoBonis/go-otel-agent/helper"
	"github.com/RodolfoBonis/go-otel-agent/integration/ginmiddleware"
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		// Provide the observability agent with custom config
		fxmodule.ProvideWithConfiguration(
			otelagent.WithServiceName("example-fx-app"),
			otelagent.WithServiceNamespace("examples"),
			otelagent.WithServiceVersion("1.0.0"),
			otelagent.WithInsecure(true),
		),

		// Provide the Gin router
		fx.Provide(newRouter),

		// Start the HTTP server
		fx.Invoke(startServer),
	)

	app.Run()
}

func newRouter(agent *otelagent.Agent, log logger.Logger) *gin.Engine {
	r := gin.Default()

	// Observability middleware
	r.Use(ginmiddleware.New(agent, "example-fx-app"))

	// Health probes
	r.GET("/health", ginmiddleware.HealthHandler(agent))
	r.GET("/ready", ginmiddleware.ReadinessHandler(agent))

	// API routes
	r.GET("/api/v1/hello", func(c *gin.Context) {
		ctx := c.Request.Context()

		log.Info(ctx, "Hello endpoint called", logger.Fields{
			"remote_addr": c.ClientIP(),
		})

		helper.Count(ctx, "api.hello.calls", 1, nil)

		c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
	})

	return r
}

func startServer(lc fx.Lifecycle, r *gin.Engine, log logger.Logger) {
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info(ctx, "Starting HTTP server on :8080")
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Error(ctx, "HTTP server error", logger.Fields{"error": err.Error()})
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info(ctx, "Shutting down HTTP server")
			return srv.Shutdown(ctx)
		},
	})
}
