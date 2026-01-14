package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/config"
	"liberation-guardian/internal/webhook"
	"liberation-guardian/pkg/types"
)

func TestWebhookReceiver(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests

	cfg := &config.Config{
		Core: config.CoreConfig{Port: 8080},
		Integrations: config.IntegrationsConfig{
			Observability: config.ObservabilityConfig{
				Sentry: config.SentryConfig{Enabled: true},
			},
		},
	}

	eventChan := make(chan *types.LiberationGuardianEvent, 10)
	receiver := webhook.NewReceiver(cfg, logger, eventChan)

	router := gin.New()
	receiver.SetupRoutes(router)

	t.Run("Universal webhook endpoint exists", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/webhook/", bytes.NewBuffer([]byte(`{"test": "data"}`)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should not return 404
		if w.Code == 404 {
			t.Errorf("Universal webhook endpoint not found")
		}
	})

	t.Run("Sentry webhook endpoint exists", func(t *testing.T) {
		sentryPayload := `{
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
		}`

		req, err := http.NewRequest("POST", "/webhook/sentry", bytes.NewBuffer([]byte(sentryPayload)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should accept webhook
		if w.Code >= 500 {
			t.Errorf("Sentry webhook failed with status %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GitHub webhook endpoint exists", func(t *testing.T) {
		githubPayload := `{
			"action": "completed",
			"workflow_run": {
				"name": "CI",
				"conclusion": "failure"
			},
			"repository": {"name": "test-repo"}
		}`

		req, err := http.NewRequest("POST", "/webhook/github", bytes.NewBuffer([]byte(githubPayload)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "workflow_run")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should accept webhook
		if w.Code >= 500 {
			t.Errorf("GitHub webhook failed with status %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Health check works", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Health check failed with status %d", w.Code)
		}
	})
}

func TestWebhookProcessors(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	t.Run("Sentry processor handles valid payload", func(t *testing.T) {
		processor := webhook.NewSentryProcessor(logger)

		payload := []byte(`{
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
		}`)

		headers := http.Header{}
		event, err := processor.ProcessWebhook(payload, headers)

		if err != nil {
			t.Errorf("Sentry processor failed: %v", err)
		}

		if event == nil {
			t.Error("Sentry processor returned nil event")
		}

		if event != nil {
			if event.Source != "sentry" {
				t.Errorf("Expected source 'sentry', got '%s'", event.Source)
			}
			if event.Title != "Test Error" {
				t.Errorf("Expected title 'Test Error', got '%s'", event.Title)
			}
		}
	})

	t.Run("GitHub processor handles workflow event", func(t *testing.T) {
		processor := webhook.NewGitHubProcessor(logger)

		payload := []byte(`{
			"action": "completed",
			"workflow_run": {
				"name": "CI",
				"conclusion": "failure"
			},
			"repository": {"name": "test-repo"}
		}`)

		headers := http.Header{}
		headers.Set("X-GitHub-Event", "workflow_run")

		event, err := processor.ProcessWebhook(payload, headers)

		if err != nil {
			t.Errorf("GitHub processor failed: %v", err)
		}

		if event == nil {
			t.Error("GitHub processor returned nil event")
		}

		if event != nil {
			if event.Source != "github" {
				t.Errorf("Expected source 'github', got '%s'", event.Source)
			}
			if event.Type != "workflow_run" {
				t.Errorf("Expected type 'workflow_run', got '%s'", event.Type)
			}
		}
	})
}
