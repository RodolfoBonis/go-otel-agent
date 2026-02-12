package main

import (
	"context"
	"log"
	"net/http"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/RodolfoBonis/go-otel-agent/helper"
	"github.com/RodolfoBonis/go-otel-agent/integration/ginmiddleware"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	agent := otelagent.NewAgent(
		otelagent.WithServiceName("example-gin-api"),
		otelagent.WithServiceNamespace("examples"),
		otelagent.WithServiceVersion("1.0.0"),
		otelagent.WithInsecure(true),
		otelagent.WithRouteExclusions(otelagent.RouteExclusionConfig{
			ExactPaths: []string{"/health", "/metrics"},
		}),
	)

	ctx := context.Background()
	if err := agent.Init(ctx); err != nil {
		log.Fatal(err)
	}
	defer agent.Shutdown(ctx)

	r := gin.Default()

	// Add observability middleware
	r.Use(ginmiddleware.New(agent, "example-gin-api"))

	// Health endpoint (excluded from tracing)
	r.GET("/health", ginmiddleware.HealthHandler(agent))
	r.GET("/ready", ginmiddleware.ReadinessHandler(agent))

	// API endpoints (automatically traced)
	r.GET("/api/v1/users", listUsers(agent))
	r.GET("/api/v1/users/:id", getUser(agent))

	log.Println("Starting server on :8080")
	r.Run(":8080")
}

func listUsers(agent *otelagent.Agent) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Add custom span attributes
		helper.SetSpanAttributes(ctx,
			attribute.String("endpoint", "list-users"),
		)

		// Record business metric
		helper.Count(ctx, "api.users.list", 1, &helper.MetricOptions{
			Component: "users",
		})

		c.JSON(http.StatusOK, gin.H{
			"users": []gin.H{
				{"id": "1", "name": "Alice"},
				{"id": "2", "name": "Bob"},
			},
		})
	}
}

func getUser(agent *otelagent.Agent) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID := c.Param("id")

		// Trace a sub-operation
		user, err := helper.TraceFunctionWithResult(ctx, agent, "fetch-user",
			func(ctx context.Context) (map[string]string, error) {
				return map[string]string{
					"id":   userID,
					"name": "Alice",
				}, nil
			},
			&helper.SpanOptions{
				Component: "users",
				Attributes: []attribute.KeyValue{
					attribute.String("user.id", userID),
				},
			},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}
