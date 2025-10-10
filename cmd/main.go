package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/ai"
	"liberation-guardian/internal/config"
	"liberation-guardian/internal/events"
	"liberation-guardian/internal/health"
	"liberation-guardian/internal/webhook"
	"liberation-guardian/pkg/types"
)

var (
	configPath = flag.String("config", "liberation-guardian.yml", "Path to configuration file")
	envFile    = flag.String("env", ".env", "Path to environment file")
)

func main() {
	flag.Parse()

	// Load environment variables
	if err := godotenv.Load(*envFile); err != nil {
		fmt.Printf("Warning: Could not load env file %s: %v\n", *envFile, err)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger := setupLogger(cfg.Core.LogLevel)
	logger.Infof("Starting Liberation Guardian %s", cfg.Core.Name)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create event channel for processing pipeline
	eventChan := make(chan *types.LiberationGuardianEvent, 1000)

	// Initialize AI client
	aiClient := ai.NewLiberationAIClient(cfg, logger)

	// Initialize event processor (integrates with existing event system)
	eventProcessor, err := events.NewProcessor(cfg, logger, aiClient)
	if err != nil {
		logger.Fatalf("Failed to create event processor: %v", err)
	}

	// Initialize webhook receiver
	webhookReceiver := webhook.NewReceiver(cfg, logger, eventChan)

	// Initialize health checker
	healthChecker := health.NewChecker(cfg, logger, aiClient)

	// Setup HTTP router
	router := setupRouter(cfg, logger, webhookReceiver, healthChecker)

	// Start event processing pipeline
	go runEventProcessor(ctx, logger, eventProcessor, eventChan)

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Core.Port),
		Handler: router,
	}

	go func() {
		logger.Infof("Starting HTTP server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Received shutdown signal, gracefully stopping...")

	// Cancel context to stop event processing
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("Liberation Guardian stopped")
}

// setupLogger configures the application logger
func setupLogger(level string) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set formatter
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	return logger
}

// setupRouter configures the HTTP router
func setupRouter(cfg *config.Config, logger *logrus.Logger, webhookReceiver *webhook.Receiver, healthChecker *health.Checker) *gin.Engine {
	// Set Gin mode based on environment
	if cfg.Core.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(loggingMiddleware(logger))

	// Health check endpoints
	router.GET("/health", healthChecker.HealthCheck)
	router.GET("/ready", healthChecker.ReadinessCheck)

	// Webhook endpoints
	webhookReceiver.SetupRoutes(router)

	// Admin/status endpoints
	api := router.Group("/api/v1")
	{
		api.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"service":     "liberation-guardian",
				"version":     "1.0.0",
				"environment": cfg.Core.Environment,
				"uptime":      time.Since(time.Now()).String(),
			})
		})
	}

	return router
}

// runEventProcessor runs the main event processing pipeline
func runEventProcessor(ctx context.Context, logger *logrus.Logger, processor *events.Processor, eventChan <-chan *types.LiberationGuardianEvent) {
	logger.Info("Starting event processing pipeline")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Event processor shutting down")
			return
		case event := <-eventChan:
			if event == nil {
				continue
			}

			// Process event asynchronously
			go func(e *types.LiberationGuardianEvent) {
				if err := processor.ProcessEvent(ctx, e); err != nil {
					logger.Errorf("Failed to process event %s: %v", e.ID, err)
				}
			}(event)
		}
	}
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}

// loggingMiddleware adds request logging
func loggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request details
		end := time.Now()
		latency := end.Sub(start)

		if raw != "" {
			path = path + "?" + raw
		}

		logger.WithFields(logrus.Fields{
			"status_code": c.Writer.Status(),
			"method":      c.Request.Method,
			"path":        path,
			"ip":          c.ClientIP(),
			"latency":     latency,
			"user_agent":  c.Request.UserAgent(),
		}).Info("HTTP Request")
	})
}
