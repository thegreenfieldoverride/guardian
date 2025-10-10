package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/ai"
	"liberation-guardian/internal/config"
)

// Checker handles health and readiness checks
type Checker struct {
	config    *config.Config
	logger    *logrus.Logger
	aiClient  ai.AIClient
	startTime time.Time
}

// NewChecker creates a new health checker
func NewChecker(cfg *config.Config, logger *logrus.Logger, aiClient ai.AIClient) *Checker {
	return &Checker{
		config:    cfg,
		logger:    logger,
		aiClient:  aiClient,
		startTime: time.Now(),
	}
}

// HealthCheck performs a basic health check
func (hc *Checker) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := gin.H{
		"service":   "liberation-guardian",
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(hc.startTime).String(),
		"version":   "1.0.0",
	}

	// Check AI client health
	if hc.aiClient != nil {
		aiHealthy := hc.aiClient.IsHealthy(ctx)
		status["ai_client_healthy"] = aiHealthy
		if !aiHealthy {
			status["status"] = "degraded"
		}
	}

	// Determine HTTP status code
	httpStatus := http.StatusOK
	if status["status"] == "degraded" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, status)
}

// ReadinessCheck performs a readiness check
func (hc *Checker) ReadinessCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := gin.H{
		"service":   "liberation-guardian",
		"ready":     true,
		"timestamp": time.Now(),
	}

	checks := gin.H{}

	// Check AI client
	if hc.aiClient != nil {
		aiHealthy := hc.aiClient.IsHealthy(ctx)
		checks["ai_client"] = aiHealthy
		if !aiHealthy {
			status["ready"] = false
		}
	}

	status["checks"] = checks

	// Determine HTTP status code
	httpStatus := http.StatusOK
	if !status["ready"].(bool) {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, status)
}
