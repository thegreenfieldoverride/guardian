package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/ai"
	"liberation-guardian/internal/config"
	"liberation-guardian/internal/health"
	"liberation-guardian/internal/webhook"
	"liberation-guardian/pkg/types"
)

func TestFullSystem(t *testing.T) {
	// Setup complete system like main.go
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests

	cfg := &config.Config{
		Core: config.CoreConfig{
			Port:        8080,
			Environment: "test",
		},
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Integrations: config.IntegrationsConfig{
			Observability: config.ObservabilityConfig{
				Sentry:  config.SentryConfig{Enabled: true},
				Grafana: config.GrafanaConfig{Enabled: true},
			},
			SourceControl: config.SourceControlConfig{
				GitHub: config.GitHubConfig{Enabled: true},
			},
		},
		AIProviders: map[string]config.AIProviderConfig{
			"triage_agent": {
				Provider:    "anthropic",
				Model:       "claude-3-sonnet",
				APIKeyEnv:   "ANTHROPIC_API_KEY",
				MaxTokens:   4000,
				Temperature: 0.1,
			},
		},
	}

	// Create components
	eventChan := make(chan *types.LiberationGuardianEvent, 10)
	aiClient := ai.NewLiberationAIClient(cfg, logger)
	webhookReceiver := webhook.NewReceiver(cfg, logger, eventChan)
	healthChecker := health.NewChecker(cfg, logger, aiClient)

	// Setup router like main.go
	router := gin.New()
	router.Use(gin.Recovery())

	// Health endpoints
	router.GET("/health", healthChecker.HealthCheck)
	router.GET("/ready", healthChecker.ReadinessCheck)

	// Webhook endpoints
	webhookReceiver.SetupRoutes(router)

	// Status endpoint
	api := router.Group("/api/v1")
	api.GET("/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service":     "liberation-guardian",
			"version":     "1.0.0",
			"environment": cfg.Core.Environment,
			"uptime":      time.Since(time.Now()).String(),
		})
	})

	t.Run("Health check works", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Health check failed with status %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Ready check works", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/ready", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 && w.Code != 503 {
			t.Errorf("Ready check failed with unexpected status %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Status endpoint works", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/status", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Status endpoint failed with status %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Webhook endpoints accept requests", func(t *testing.T) {
		tests := []struct {
			endpoint string
			payload  string
			headers  map[string]string
		}{
			{
				endpoint: "/webhook/sentry",
				payload: `{
					"action": "created",
					"data": {
						"issue": {
							"id": "123",
							"title": "Test Error",
							"level": "error",
							"message": "Something went wrong",
							"firstSeen": "2023-01-01T00:00:00Z",
							"project": {"name": "test-project", "slug": "test"}
						}
					}
				}`,
				headers: map[string]string{"Content-Type": "application/json"},
			},
			{
				endpoint: "/webhook/github",
				payload: `{
					"action": "completed",
					"workflow_run": {
						"name": "CI",
						"conclusion": "failure"
					},
					"repository": {"name": "test-repo"}
				}`,
				headers: map[string]string{
					"Content-Type":   "application/json",
					"X-GitHub-Event": "workflow_run",
				},
			},
		}

		for _, test := range tests {
			req, err := http.NewRequest("POST", test.endpoint, bytes.NewBuffer([]byte(test.payload)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
			for k, v := range test.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should accept webhook (200) or handle gracefully (4xx)
			if w.Code >= 500 {
				t.Errorf("Webhook %s failed with status %d: %s", test.endpoint, w.Code, w.Body.String())
			}
		}
	})

	t.Run("Universal webhook detects sources", func(t *testing.T) {
		// Test auto-detection of Sentry webhook
		sentryPayload := `{
			"action": "created",
			"data": {
				"issue": {
					"id": "123",
					"title": "Test Error",
					"level": "error"
				}
			}
		}`

		req, err := http.NewRequest("POST", "/webhook/", bytes.NewBuffer([]byte(sentryPayload)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Sentry/1.0")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should handle auto-detection
		if w.Code >= 500 {
			t.Errorf("Universal webhook failed with status %d: %s", w.Code, w.Body.String())
		}
	})
}
